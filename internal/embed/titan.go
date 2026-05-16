package embed

import (
	"context"
	"errors"
	"fmt"

	"github.com/cax/fsdtrace/internal/db"
)

// TitanModelID is the default Bedrock embedding model.
const TitanModelID = "amazon.titan-embed-text-v2:0"

// TitanEmbedRequest is the body for /model/amazon.titan-embed-text-v2:0/invoke.
// Titan v2 returns a float[1024] embedding.
type TitanEmbedRequest struct {
	InputText string `json:"inputText"`
	// Dimensions: omit to use Titan v2 default (1024).
	Dimensions int  `json:"dimensions,omitempty"`
	Normalize  bool `json:"normalize,omitempty"`
}

// TitanEmbedResponse is the Titan v2 response shape.
type TitanEmbedResponse struct {
	Embedding           []float32 `json:"embedding"`
	InputTextTokenCount int       `json:"inputTextTokenCount"`
}

// Embedder is the minimal interface the atomizer and matcher depend on.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Model() string
}

// TitanEmbedder calls Titan v2 via BedrockClient. Implements Embedder.
type TitanEmbedder struct {
	c     *BedrockClient
	model string
}

// NewTitanEmbedder builds an embedder. If model is "" the default
// TitanModelID is used.
func NewTitanEmbedder(c *BedrockClient, model string) *TitanEmbedder {
	if model == "" {
		model = TitanModelID
	}
	return &TitanEmbedder{c: c, model: model}
}

// Embed returns a 1024-dim vector for text.
func (t *TitanEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, errors.New("titan: empty input text")
	}
	var resp TitanEmbedResponse
	if err := t.c.Invoke(ctx, t.model, TitanEmbedRequest{InputText: text}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Embedding) != db.EmbeddingDim {
		return nil, fmt.Errorf("titan: unexpected dim %d (want %d)",
			len(resp.Embedding), db.EmbeddingDim)
	}
	return resp.Embedding, nil
}

// Model returns the underlying Bedrock model identifier.
func (t *TitanEmbedder) Model() string { return t.model }
