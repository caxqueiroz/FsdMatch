package embed

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
)

func TestBedrockEmbedderDefaultsToTitan(t *testing.T) {
	gotPath, gotBody, emb := runEmbedRequest(t, "", PurposeDocument)

	if gotPath != "/model/amazon.titan-embed-text-v2:0/invoke" {
		t.Fatalf("path = %q", gotPath)
	}
	var req TitanEmbedRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatal(err)
	}
	if req.InputText != "find orders" {
		t.Fatalf("inputText = %q", req.InputText)
	}
	if emb.Model() != TitanModelID {
		t.Fatalf("model = %q", emb.Model())
	}
}

func TestCohereDocumentEmbedderUsesSearchDocumentInputType(t *testing.T) {
	gotPath, gotBody, emb := runEmbedRequest(t, "cohere.embed-english-v3", PurposeDocument)

	if gotPath != "/model/cohere.embed-english-v3/invoke" {
		t.Fatalf("path = %q", gotPath)
	}
	var req CohereEmbedRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatal(err)
	}
	if req.InputType != "search_document" {
		t.Fatalf("input_type = %q", req.InputType)
	}
	if len(req.Texts) != 1 || req.Texts[0] != "find orders" {
		t.Fatalf("texts = %#v", req.Texts)
	}
	if req.Truncate != "END" {
		t.Fatalf("truncate = %q", req.Truncate)
	}
	if req.OutputDimension != 0 {
		t.Fatalf("output_dimension = %d", req.OutputDimension)
	}
	if emb.Model() != "cohere.embed-english-v3#input_type=search_document" {
		t.Fatalf("model cache key = %q", emb.Model())
	}
}

func TestCohereQueryEmbedderUsesSearchQueryInputTypeAndV4Dimension(t *testing.T) {
	gotPath, gotBody, emb := runEmbedRequest(t, "cohere.embed-v4", PurposeQuery)

	if gotPath != "/model/cohere.embed-v4/invoke" {
		t.Fatalf("path = %q", gotPath)
	}
	var req CohereEmbedRequest
	if err := json.Unmarshal(gotBody, &req); err != nil {
		t.Fatal(err)
	}
	if req.InputType != "search_query" {
		t.Fatalf("input_type = %q", req.InputType)
	}
	if req.OutputDimension != db.EmbeddingDim {
		t.Fatalf("output_dimension = %d", req.OutputDimension)
	}
	if req.Truncate != "RIGHT" {
		t.Fatalf("truncate = %q", req.Truncate)
	}
	if emb.Model() != "cohere.embed-v4#input_type=search_query#dim=1024" {
		t.Fatalf("model cache key = %q", emb.Model())
	}
}

func TestCohereDocumentAndQueryCacheKeysDiffer(t *testing.T) {
	c, err := NewClient("https://bedrock.local")
	if err != nil {
		t.Fatal(err)
	}
	doc := NewBedrockEmbedder(c, "cohere.embed-english-v3", PurposeDocument)
	query := NewBedrockEmbedder(c, "cohere.embed-english-v3", PurposeQuery)

	if CacheKey(doc.Model(), "same text") == CacheKey(query.Model(), "same text") {
		t.Fatal("document and query embeddings must not share a cache key")
	}
}

func runEmbedRequest(t *testing.T, model string, purpose Purpose) (string, []byte, Embedder) {
	t.Helper()
	var (
		gotPath string
		gotBody []byte
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		gotBody = body

		v := make([]float32, db.EmbeddingDim)
		v[0] = 1
		var payload []byte
		switch model {
		case "cohere.embed-english-v3", "cohere.embed-v4":
			payload, _ = json.Marshal(CohereEmbedResponse{Embeddings: mustRawMessage(t, [][]float32{v})})
		default:
			payload, _ = json.Marshal(TitanEmbedResponse{Embedding: v, InputTextTokenCount: 2})
		}
		_, _ = w.Write(payload)
	}))
	t.Cleanup(ts.Close)

	c, err := NewClient(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	emb := NewBedrockEmbedder(c, model, purpose)
	if _, err := emb.Embed(context.Background(), "find orders"); err != nil {
		t.Fatalf("embed: %v", err)
	}
	return gotPath, gotBody, emb
}

func mustRawMessage(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
