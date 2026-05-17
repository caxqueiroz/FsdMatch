package embed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
)

func TestOpenAIEmbedderUsesEmbeddingsAPI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["model"]; got != OpenAIEmbeddingModelID {
			t.Fatalf("model = %v", got)
		}
		if got := body["input"]; got != "hello world" {
			t.Fatalf("input = %v", got)
		}
		if got := body["dimensions"]; got != float64(db.EmbeddingDim) {
			t.Fatalf("dimensions = %v", got)
		}
		_, _ = w.Write([]byte(`{
			"object":"list",
			"model":"text-embedding-3-large",
			"data":[{"object":"embedding","index":0,"embedding":[1.25,2.5,3.75]}]
		}`))
	}))
	t.Cleanup(ts.Close)

	emb, err := NewOpenAIEmbedder("test-key", OpenAIEmbeddingModelID, PurposeDocument,
		WithOpenAIEmbedderBaseURL(ts.URL),
		WithOpenAIEmbedderHTTPClient(ts.Client()))
	if err != nil {
		t.Fatal(err)
	}
	got, err := emb.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != 1.25 || got[2] != 3.75 {
		t.Fatalf("embedding = %+v", got)
	}
	if want := "openai:text-embedding-3-large#dim=1024"; emb.Model() != want {
		t.Fatalf("Model() = %q, want %q", emb.Model(), want)
	}
}
