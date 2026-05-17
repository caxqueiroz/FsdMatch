package cli

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/llm"
)

const (
	// EnvBedrockBaseURL is the env variable read for the Bedrock route.
	// The CLI never hardcodes the AWS endpoint (CLAUDE.md hard constraint).
	EnvBedrockBaseURL = "BEDROCK_BASE_URL"

	// EnvOpenAIAPIKey is the direct OpenAI API key used by provider=openai.
	EnvOpenAIAPIKey = "OPENAI_API_KEY" // #nosec G101 -- env var name, not a credential.

	openAIJudgmentMaxTokens = 12000
	openAIJudgmentBatchSize = 8
)

func resolveProvider(cfg appConfig, flagValue string, flagChanged bool) (appConfig, string, error) {
	provider, err := cfg.providerName(flagValue, flagChanged)
	if err != nil {
		return cfg, "", err
	}
	return cfg.withProvider(provider), provider, nil
}

func newGenerator(provider, cassettePath string, cfg appConfig) (llm.Generator, error) {
	switch provider {
	case ProviderBedrock:
		bedrock, err := newBedrockClient(cassettePath, cfg)
		if err != nil {
			return nil, err
		}
		return llm.NewBedrockGenerator(bedrock), nil
	case ProviderOpenAI:
		if cassettePath != "" {
			return nil, fmt.Errorf("--cassette is only supported with --provider %s", ProviderBedrock)
		}
		return newOpenAIGenerator(cfg)
	default:
		return nil, fmt.Errorf("unknown provider %q (want bedrock|openai)", provider)
	}
}

func newEmbedder(provider, cassettePath string, cfg appConfig, model string, purpose embed.Purpose) (embed.Embedder, error) {
	switch provider {
	case ProviderBedrock:
		bedrock, err := newBedrockClient(cassettePath, cfg)
		if err != nil {
			return nil, err
		}
		return embed.NewBedrockEmbedder(bedrock, model, purpose), nil
	case ProviderOpenAI:
		if cassettePath != "" {
			return nil, fmt.Errorf("--cassette is only supported with --provider %s", ProviderBedrock)
		}
		return newOpenAIEmbedder(cfg, model, purpose)
	default:
		return nil, fmt.Errorf("unknown provider %q (want bedrock|openai)", provider)
	}
}

// newBedrockClient builds a BedrockClient honouring the env variable
// and an optional cassette file.
func newBedrockClient(cassettePath string, cfg appConfig) (*embed.BedrockClient, error) {
	if cassettePath != "" {
		cas, err := embed.LoadCassette(cassettePath)
		if err != nil {
			return nil, err
		}
		// Cassette plays back URL-independently; any non-empty base is fine.
		return embed.NewClient("https://cassette.local",
			embed.WithHTTPClient(cas.HTTPClient()))
	}
	base := cfg.bedrockURL()
	if base == "" {
		return nil, fmt.Errorf("%s is unset; set it to the KrakenD route or pass --cassette",
			EnvBedrockBaseURL)
	}
	return embed.NewClient(base,
		embed.WithHTTPClient(&http.Client{Timeout: 90 * time.Second}))
}

func newOpenAIGenerator(cfg appConfig) (llm.Generator, error) {
	apiKey := os.Getenv(EnvOpenAIAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s is unset; set it to use --provider %s",
			EnvOpenAIAPIKey, ProviderOpenAI)
	}
	opts := []llm.OpenAIOption{}
	if baseURL := cfg.openAIURL(); baseURL != "" {
		opts = append(opts, llm.WithOpenAIBaseURL(baseURL))
	}
	return llm.NewOpenAIClient(apiKey, opts...)
}

func newOpenAIEmbedder(cfg appConfig, model string, purpose embed.Purpose) (embed.Embedder, error) {
	apiKey := os.Getenv(EnvOpenAIAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s is unset; set it to use --provider %s",
			EnvOpenAIAPIKey, ProviderOpenAI)
	}
	opts := []embed.OpenAIEmbedderOption{}
	if baseURL := cfg.openAIURL(); baseURL != "" {
		opts = append(opts, embed.WithOpenAIEmbedderBaseURL(baseURL))
	}
	return embed.NewOpenAIEmbedder(apiKey, model, purpose, opts...)
}
