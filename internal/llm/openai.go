package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// HTTPDoer is the subset of *http.Client used by OpenAIClient.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// OpenAIClient invokes the OpenAI Responses API.
type OpenAIClient struct {
	apiKey  string
	baseURL string
	http    HTTPDoer
}

// OpenAIOption configures an OpenAIClient.
type OpenAIOption func(*OpenAIClient)

// WithOpenAIBaseURL overrides the default OpenAI API base URL. Tests use
// this to point at httptest; production normally leaves it unset.
func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(c *OpenAIClient) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithOpenAIHTTPClient overrides the default HTTP client.
func WithOpenAIHTTPClient(h HTTPDoer) OpenAIOption {
	return func(c *OpenAIClient) { c.http = h }
}

// NewOpenAIClient constructs a Responses API client.
func NewOpenAIClient(apiKey string, opts ...OpenAIOption) (*OpenAIClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("openai: API key is empty")
	}
	c := &OpenAIClient{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: defaultOpenAIBaseURL,
		http:    &http.Client{Timeout: 90 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, errors.New("openai: base URL is empty")
	}
	if _, err := url.ParseRequestURI(c.baseURL); err != nil {
		return nil, fmt.Errorf("openai: parsing base URL %q: %w", c.baseURL, err)
	}
	return c, nil
}

// Generate calls POST /responses and returns the response text.
func (c *OpenAIClient) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = DefaultOpenAIModel
	}
	body := openAIResponseRequest{
		Model:           model,
		Instructions:    req.System,
		Input:           req.User,
		MaxOutputTokens: req.MaxTokens,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("openai: marshal response request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai: responses: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai: read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("openai: responses HTTP %d: %s", resp.StatusCode, truncateString(string(respBody), 512))
	}
	var parsed openAIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("openai: decode response: %w; body=%s", err, truncateString(string(respBody), 256))
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("openai: %s: %s", parsed.Error.Code, parsed.Error.Message)
	}
	if parsed.Status == "incomplete" || parsed.IncompleteDetails != nil {
		reason := "unknown"
		if parsed.IncompleteDetails != nil && parsed.IncompleteDetails.Reason != "" {
			reason = parsed.IncompleteDetails.Reason
		}
		return "", &IncompleteError{Provider: "openai", Reason: reason}
	}
	text := strings.TrimSpace(parsed.OutputText)
	if text == "" {
		text = strings.TrimSpace(parsed.nestedText())
	}
	if text == "" {
		return "", errors.New("openai: empty response")
	}
	return text, nil
}

type openAIResponseRequest struct {
	Model           string `json:"model"`
	Instructions    string `json:"instructions,omitempty"`
	Input           string `json:"input"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
}

type openAIResponse struct {
	Status     string `json:"status"`
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

func (r openAIResponse) nestedText() string {
	var b strings.Builder
	for _, out := range r.Output {
		for _, c := range out.Content {
			if c.Text != "" {
				b.WriteString(c.Text)
			}
		}
	}
	return b.String()
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
