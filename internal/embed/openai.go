package embed

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

	"github.com/cax/fsdtrace/internal/db"
)

const (
	// OpenAIEmbeddingModelID is the default OpenAI embedding model. It is
	// invoked with dimensions=1024 to match the existing vec0 schema.
	OpenAIEmbeddingModelID = "text-embedding-3-large"

	defaultOpenAIBaseURL = "https://api.openai.com/v1"
)

// OpenAIEmbedder embeds text through OpenAI's /embeddings endpoint.
type OpenAIEmbedder struct {
	apiKey     string
	model      string
	cacheModel string
	baseURL    string
	http       HTTPDoer
}

// OpenAIEmbedderOption configures an OpenAIEmbedder.
type OpenAIEmbedderOption func(*OpenAIEmbedder)

// WithOpenAIEmbedderBaseURL overrides the default OpenAI API base URL.
func WithOpenAIEmbedderBaseURL(baseURL string) OpenAIEmbedderOption {
	return func(e *OpenAIEmbedder) { e.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithOpenAIEmbedderHTTPClient overrides the default HTTP client.
func WithOpenAIEmbedderHTTPClient(h HTTPDoer) OpenAIEmbedderOption {
	return func(e *OpenAIEmbedder) { e.http = h }
}

// NewOpenAIEmbedder constructs an OpenAI embedding adapter.
func NewOpenAIEmbedder(apiKey, model string, _ Purpose, opts ...OpenAIEmbedderOption) (*OpenAIEmbedder, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("openai embedding: API key is empty")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = OpenAIEmbeddingModelID
	}
	e := &OpenAIEmbedder{
		apiKey:     strings.TrimSpace(apiKey),
		model:      model,
		cacheModel: fmt.Sprintf("openai:%s#dim=%d", model, db.EmbeddingDim),
		baseURL:    defaultOpenAIBaseURL,
		http:       &http.Client{Timeout: 90 * time.Second},
	}
	for _, o := range opts {
		o(e)
	}
	if strings.TrimSpace(e.baseURL) == "" {
		return nil, errors.New("openai embedding: base URL is empty")
	}
	if _, err := url.ParseRequestURI(e.baseURL); err != nil {
		return nil, fmt.Errorf("openai embedding: parsing base URL %q: %w", e.baseURL, err)
	}
	return e, nil
}

// Embed returns a single float embedding for text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body := openAIEmbeddingRequest{
		Model:          e.model,
		Input:          text,
		Dimensions:     db.EmbeddingDim,
		EncodingFormat: "float",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai embedding: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embedding: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai embedding: read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai embedding: HTTP %d: %s", resp.StatusCode, truncate(respBody, 512))
	}
	var parsed openAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("openai embedding: decode: %w; body=%s", err, truncate(respBody, 256))
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, errors.New("openai embedding: response missing embedding")
	}
	return parsed.Data[0].Embedding, nil
}

// Model returns the cache key model string, including provider and
// dimension so OpenAI vectors never collide with Bedrock vectors.
func (e *OpenAIEmbedder) Model() string { return e.cacheModel }

type openAIEmbeddingRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Dimensions     int    `json:"dimensions"`
	EncodingFormat string `json:"encoding_format"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
		Object    string    `json:"object"`
	} `json:"data"`
	Model string `json:"model"`
}
