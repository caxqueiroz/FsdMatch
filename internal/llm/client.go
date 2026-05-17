// Package llm normalizes text-generation calls across provider APIs.
package llm

import (
	"context"
	"errors"
	"fmt"
)

const (
	// DefaultOpenAIModel is the OpenAI model used when provider=openai
	// and no task-specific model override is configured.
	DefaultOpenAIModel = "gpt-5.5"
)

// GenerateRequest is the provider-neutral request shape used by the
// atomizer and matcher prompts.
type GenerateRequest struct {
	Model     string
	System    string
	User      string
	MaxTokens int
}

// Generator produces a single text response for a prompt.
type Generator interface {
	Generate(context.Context, GenerateRequest) (string, error)
}

// IncompleteError reports that a provider stopped before producing a
// complete response, usually because max output tokens were reached.
type IncompleteError struct {
	Provider string
	Reason   string
}

func (e *IncompleteError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("%s: response incomplete", e.Provider)
	}
	return fmt.Sprintf("%s: response incomplete: %s", e.Provider, e.Reason)
}

// IsIncomplete reports whether err wraps an IncompleteError.
func IsIncomplete(err error) bool {
	var target *IncompleteError
	return errors.As(err, &target)
}
