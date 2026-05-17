package embed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cax/fsdtrace/internal/db"
)

// CohereEmbedRequest is the Bedrock request body for Cohere Embed models.
type CohereEmbedRequest struct {
	InputType       string   `json:"input_type"`
	Texts           []string `json:"texts"`
	Truncate        string   `json:"truncate,omitempty"`
	OutputDimension int      `json:"output_dimension,omitempty"`
}

// CohereEmbedResponse is the Bedrock response body for Cohere Embed models.
type CohereEmbedResponse struct {
	Embeddings json.RawMessage `json:"embeddings"`
}

// CohereEmbedder calls Cohere Embed through BedrockClient.
type CohereEmbedder struct {
	c               *BedrockClient
	modelID         string
	cacheModel      string
	inputType       string
	truncate        string
	outputDimension int
}

// NewCohereEmbedder builds a Cohere embedder for document or query usage.
func NewCohereEmbedder(c *BedrockClient, model string, purpose Purpose) *CohereEmbedder {
	inputType := cohereInputType(purpose)
	e := &CohereEmbedder{
		c:          c,
		modelID:    model,
		cacheModel: model + "#input_type=" + inputType,
		inputType:  inputType,
		truncate:   "END",
	}
	if model == "cohere.embed-v4" {
		e.truncate = "RIGHT"
		e.outputDimension = db.EmbeddingDim
		e.cacheModel = fmt.Sprintf("%s#input_type=%s#dim=%d", model, inputType, db.EmbeddingDim)
	}
	return e
}

// Embed returns a 1024-dim vector for text.
func (e *CohereEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, errors.New("cohere: empty input text")
	}
	req := CohereEmbedRequest{
		InputType:       e.inputType,
		Texts:           []string{text},
		Truncate:        e.truncate,
		OutputDimension: e.outputDimension,
	}
	var resp CohereEmbedResponse
	if err := e.c.Invoke(ctx, e.modelID, req, &resp); err != nil {
		return nil, err
	}
	v, err := resp.firstEmbedding()
	if err != nil {
		return nil, err
	}
	if len(v) != db.EmbeddingDim {
		return nil, fmt.Errorf("cohere: unexpected dim %d (want %d)",
			len(v), db.EmbeddingDim)
	}
	return v, nil
}

// Model returns a cache-safe provider configuration identifier.
func (e *CohereEmbedder) Model() string { return e.cacheModel }

func cohereInputType(purpose Purpose) string {
	if purpose == PurposeQuery {
		return "search_query"
	}
	return "search_document"
}

func (r CohereEmbedResponse) firstEmbedding() ([]float32, error) {
	if len(r.Embeddings) == 0 {
		return nil, errors.New("cohere: response missing embeddings")
	}
	var vectors [][]float32
	if err := json.Unmarshal(r.Embeddings, &vectors); err == nil {
		return firstVector(vectors)
	}

	var byType map[string][][]float32
	if err := json.Unmarshal(r.Embeddings, &byType); err != nil {
		return nil, fmt.Errorf("cohere: decode embeddings: %w", err)
	}
	vectors, ok := byType["float"]
	if !ok {
		return nil, errors.New("cohere: response missing float embeddings")
	}
	return firstVector(vectors)
}

func firstVector(vectors [][]float32) ([]float32, error) {
	if len(vectors) == 0 {
		return nil, errors.New("cohere: response has no embeddings")
	}
	if len(vectors[0]) == 0 {
		return nil, errors.New("cohere: response has empty embedding")
	}
	return vectors[0], nil
}
