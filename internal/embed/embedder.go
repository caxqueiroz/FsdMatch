package embed

import "strings"

// Purpose describes how an embedding will be used. Providers such as
// Cohere encode document and query vectors differently for retrieval.
type Purpose string

const (
	// PurposeDocument is for corpus rows stored in vec0.
	PurposeDocument Purpose = "document"
	// PurposeQuery is for lookup text used to query vec0.
	PurposeQuery Purpose = "query"
)

// NewBedrockEmbedder selects the request/response adapter required by
// model. Titan remains the default provider.
func NewBedrockEmbedder(c *BedrockClient, model string, purpose Purpose) Embedder {
	model = strings.TrimSpace(model)
	if model == "" {
		return NewTitanEmbedder(c, model)
	}
	if isCohereEmbedModel(model) {
		return NewCohereEmbedder(c, model, purpose)
	}
	return NewTitanEmbedder(c, model)
}

func isCohereEmbedModel(model string) bool {
	return strings.HasPrefix(model, "cohere.embed-")
}
