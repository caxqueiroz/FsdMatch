package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
	"github.com/cax/fsdtrace/internal/llm"
	"github.com/cax/fsdtrace/internal/match"
)

const (
	// ProviderBedrock keeps the original KrakenD -> Bedrock behavior.
	ProviderBedrock = "bedrock"
	// ProviderOpenAI uses OpenAI directly for generation and embeddings.
	ProviderOpenAI = "openai"

	// EnvProvider selects the model provider: bedrock or openai.
	EnvProvider = "FSDTRACE_PROVIDER"
	// EnvEmbeddingModel overrides the embedding model.
	EnvEmbeddingModel = "FSDTRACE_EMBEDDING_MODEL"
	// EnvAtomizerModel overrides the model used for FSD atomization.
	EnvAtomizerModel = "FSDTRACE_ATOMIZER_MODEL"
	// EnvJudgmentModel overrides the model used for matching.
	EnvJudgmentModel = "FSDTRACE_JUDGMENT_MODEL"
	// EnvRejudgeModel overrides the stronger model used for drift rejudging.
	EnvRejudgeModel = "FSDTRACE_REJUDGE_MODEL"
	// EnvOpenAIBaseURL optionally overrides the OpenAI API base URL.
	EnvOpenAIBaseURL = "FSDTRACE_OPENAI_BASE_URL"

	// DefaultRejudgeModel is the built-in fallback when no flag, env, or
	// config file value is provided.
	DefaultRejudgeModel = "anthropic.claude-opus-4-v2:0"
)

type modelKind string

const (
	modelEmbedding modelKind = "embedding"
	modelAtomizer  modelKind = "atomizer"
	modelJudgment  modelKind = "judgment"
	modelRejudge   modelKind = "rejudge"
)

type appConfig struct {
	provider string

	bedrockBaseURL string
	openAIBaseURL  string

	embeddingModel string
	atomizerModel  string
	judgmentModel  string
	rejudgeModel   string

	bedrockEmbeddingModel string
	bedrockAtomizerModel  string
	bedrockJudgmentModel  string
	bedrockRejudgeModel   string

	openAIEmbeddingModel string
	openAIAtomizerModel  string
	openAIJudgmentModel  string
	openAIRejudgeModel   string
}

func loadAppConfig(path string, getenv func(string) string) (appConfig, error) {
	cfg := appConfig{}
	if getenv == nil {
		getenv = os.Getenv
	}

	fileCfg, err := readAppConfigFile(path)
	if err != nil {
		return cfg, err
	}
	cfg = fileCfg

	cfg.provider = firstNonEmpty(getenv(EnvProvider), cfg.provider)
	cfg.bedrockBaseURL = firstNonEmpty(getenv(EnvBedrockBaseURL), cfg.bedrockBaseURL)
	cfg.openAIBaseURL = firstNonEmpty(getenv(EnvOpenAIBaseURL), cfg.openAIBaseURL)
	cfg.embeddingModel = firstNonEmpty(getenv(EnvEmbeddingModel), cfg.embeddingModel)
	cfg.atomizerModel = firstNonEmpty(getenv(EnvAtomizerModel), cfg.atomizerModel)
	cfg.judgmentModel = firstNonEmpty(getenv(EnvJudgmentModel), cfg.judgmentModel)
	cfg.rejudgeModel = firstNonEmpty(getenv(EnvRejudgeModel), cfg.rejudgeModel)
	return cfg, nil
}

func (c appConfig) model(kind modelKind, flagValue string, flagChanged bool) string {
	if flagChanged && strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	provider := c.activeProvider()
	switch kind {
	case modelEmbedding:
		if provider == ProviderOpenAI {
			return firstNonEmpty(c.embeddingModel, c.openAIEmbeddingModel, embed.OpenAIEmbeddingModelID)
		}
		return firstNonEmpty(c.embeddingModel, c.bedrockEmbeddingModel, embed.TitanModelID)
	case modelAtomizer:
		if provider == ProviderOpenAI {
			return firstNonEmpty(c.atomizerModel, c.openAIAtomizerModel, llm.DefaultOpenAIModel)
		}
		return firstNonEmpty(c.atomizerModel, c.bedrockAtomizerModel, fsd.DefaultAtomizerModel)
	case modelJudgment:
		if provider == ProviderOpenAI {
			return firstNonEmpty(c.judgmentModel, c.openAIJudgmentModel, llm.DefaultOpenAIModel)
		}
		return firstNonEmpty(c.judgmentModel, c.bedrockJudgmentModel, match.DefaultJudgmentModel)
	case modelRejudge:
		if provider == ProviderOpenAI {
			return firstNonEmpty(c.rejudgeModel, c.openAIRejudgeModel, llm.DefaultOpenAIModel)
		}
		return firstNonEmpty(c.rejudgeModel, c.bedrockRejudgeModel, DefaultRejudgeModel)
	default:
		return ""
	}
}

func (c appConfig) providerName(flagValue string, flagChanged bool) (string, error) {
	provider := c.provider
	if flagChanged && strings.TrimSpace(flagValue) != "" {
		provider = flagValue
	}
	provider = strings.ToLower(firstNonEmpty(provider, ProviderBedrock))
	switch provider {
	case ProviderBedrock, ProviderOpenAI:
		return provider, nil
	default:
		return "", fmt.Errorf("unknown provider %q (want bedrock|openai)", provider)
	}
}

func (c appConfig) withProvider(provider string) appConfig {
	c.provider = provider
	return c
}

func (c appConfig) activeProvider() string {
	provider, err := c.providerName("", false)
	if err != nil {
		return ProviderBedrock
	}
	return provider
}

func (c appConfig) bedrockURL() string {
	return strings.TrimSpace(c.bedrockBaseURL)
}

func (c appConfig) openAIURL() string {
	return strings.TrimSpace(c.openAIBaseURL)
}

func readAppConfigFile(path string) (appConfig, error) {
	if path != "" {
		return parseConfigPath(path, true)
	}
	for _, p := range defaultConfigPaths() {
		cfg, err := parseConfigPath(p, false)
		if err == nil {
			return cfg, nil
		}
		if !os.IsNotExist(err) {
			return appConfig{}, err
		}
	}
	return appConfig{}, nil
}

func defaultConfigPaths() []string {
	paths := []string{"fsdtrace.yaml"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths, filepath.Join(home, ".fsdtrace.yaml"))
	}
	return paths
}

func parseConfigPath(path string, required bool) (appConfig, error) {
	f, err := os.Open(path) // #nosec G304 -- user-selected config path
	if err != nil {
		if !required && os.IsNotExist(err) {
			return appConfig{}, err
		}
		return appConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var cfg appConfig
	section := ""
	scanner := bufio.NewScanner(f)
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := stripComment(scanner.Text())
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingSpaces(line)
		key, value, ok := splitConfigLine(strings.TrimSpace(line))
		if !ok {
			return cfg, fmt.Errorf("parse config %s:%d: expected key: value", path, lineNum)
		}
		if indent == 0 && value == "" {
			section = key
			continue
		}
		fullKey := key
		if indent > 0 && section != "" {
			fullKey = section + "." + key
		}
		setConfigValue(&cfg, fullKey, value)
	}
	if err := scanner.Err(); err != nil {
		return cfg, fmt.Errorf("scan config %s: %w", path, err)
	}
	return cfg, nil
}

func stripComment(s string) string {
	if i := strings.Index(s, "#"); i >= 0 {
		return s[:i]
	}
	return s
}

func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

func splitConfigLine(s string) (key, value string, ok bool) {
	k, v, found := strings.Cut(s, ":")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(k)
	value = strings.TrimSpace(v)
	value = strings.Trim(value, `"'`)
	return key, value, key != ""
}

func setConfigValue(cfg *appConfig, key, value string) {
	value = strings.TrimSpace(value)
	switch key {
	case "provider", "model_provider":
		cfg.provider = value
	case "bedrock.base_url":
		cfg.bedrockBaseURL = value
	case "openai.base_url":
		cfg.openAIBaseURL = value
	case "embedding_model":
		cfg.embeddingModel = value
	case "atomizer_model":
		cfg.atomizerModel = value
	case "judgment_model":
		cfg.judgmentModel = value
	case "rejudge_model":
		cfg.rejudgeModel = value
	case "bedrock.embedding_model":
		cfg.bedrockEmbeddingModel = value
	case "bedrock.atomizer_model":
		cfg.bedrockAtomizerModel = value
	case "bedrock.judgment_model":
		cfg.bedrockJudgmentModel = value
	case "bedrock.rejudge_model":
		cfg.bedrockRejudgeModel = value
	case "openai.embedding_model":
		cfg.openAIEmbeddingModel = value
	case "openai.atomizer_model":
		cfg.openAIAtomizerModel = value
	case "openai.judgment_model":
		cfg.openAIJudgmentModel = value
	case "openai.rejudge_model":
		cfg.openAIRejudgeModel = value
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
