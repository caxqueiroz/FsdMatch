package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cax/fsdtrace/internal/embed"
)

// BedrockAnthropicVersion is the required `anthropic_version` field for
// the Bedrock-hosted Claude Messages API.
const BedrockAnthropicVersion = "bedrock-2023-05-31"

// BedrockGenerator calls Anthropic models through the existing Bedrock
// gateway client.
type BedrockGenerator struct {
	c *embed.BedrockClient
}

// NewBedrockGenerator adapts a BedrockClient to the provider-neutral
// Generator interface.
func NewBedrockGenerator(c *embed.BedrockClient) *BedrockGenerator {
	return &BedrockGenerator{c: c}
}

// Generate invokes the Anthropic Messages envelope used by Bedrock.
func (g *BedrockGenerator) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	if g == nil || g.c == nil {
		return "", errors.New("bedrock generator: nil client")
	}
	if strings.TrimSpace(req.Model) == "" {
		return "", errors.New("bedrock generator: model is required")
	}
	body := bedrockMessage{
		AnthropicVersion: BedrockAnthropicVersion,
		MaxTokens:        req.MaxTokens,
		System:           req.System,
		Messages: []bedrockMessageEntry{
			{Role: "user", Content: req.User},
		},
	}
	var resp bedrockMessageResponse
	if err := g.c.Invoke(ctx, req.Model, body, &resp); err != nil {
		return "", err
	}
	text := joinBedrockResponseText(resp)
	if text == "" {
		return "", fmt.Errorf("bedrock generator: empty response from %s", req.Model)
	}
	return text, nil
}

type bedrockMessage struct {
	AnthropicVersion string                `json:"anthropic_version"`
	MaxTokens        int                   `json:"max_tokens"`
	System           string                `json:"system,omitempty"`
	Messages         []bedrockMessageEntry `json:"messages"`
}

type bedrockMessageEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

func joinBedrockResponseText(r bedrockMessageResponse) string {
	var b strings.Builder
	for _, c := range r.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return strings.TrimSpace(b.String())
}
