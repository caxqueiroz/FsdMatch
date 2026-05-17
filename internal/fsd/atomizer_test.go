package fsd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/llm"
)

// fakeBedrock is an httptest server that returns deterministic responses
// keyed by Bedrock model path. It records call counts so tests can
// assert idempotency.
type fakeBedrock struct {
	atomizeCalls atomic.Int64
	embedCalls   atomic.Int64
}

func (f *fakeBedrock) handler(t *testing.T) http.Handler {
	t.Helper()
	// Match the per-chunk anchor that BuildAtomizerUserMessage embeds.
	// The system prompt also contains "FR-042" as an example so a naive
	// unscoped search would collapse everything.
	anchorRe := regexp.MustCompile(`Anchor:\s*(\bFR(?:-[A-Z0-9]+)*-\d+\b)`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/model/amazon.titan-embed-text-v2:0/invoke"):
			f.embedCalls.Add(1)
			emb := make([]float32, db.EmbeddingDim)
			emb[0] = 1
			payload, _ := json.Marshal(embed.TitanEmbedResponse{Embedding: emb, InputTextTokenCount: 1})
			_, _ = w.Write(payload)

		case strings.Contains(r.URL.Path, "/invoke"):
			f.atomizeCalls.Add(1)
			// Echo back a synthetic FR JSON sourced from the anchor in the prompt.
			anchor := "FR-000"
			if m := anchorRe.FindStringSubmatch(string(body)); len(m) > 1 {
				anchor = m[1]
			}
			frJSON := fmt.Sprintf(`{
				"id":           %q,
				"title":        "Title for %s",
				"description":  "Description for %s",
				"acceptance":   ["criterion 1","criterion 2"],
				"actor":        "user",
				"inputs":       ["input"],
				"outputs":      ["output"],
				"side_effects": [],
				"non_functional": []
			}`, anchor, anchor, anchor)
			resp := map[string]any{
				"content":     []map[string]any{{"type": "text", "text": frJSON}},
				"stop_reason": "end_turn",
			}
			payload, _ := json.Marshal(resp)
			_, _ = w.Write(payload)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	})
}

func setupAtomizer(t *testing.T, fb *fakeBedrock) (*Atomizer, *db.DB, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(fb.handler(t))
	t.Cleanup(ts.Close)

	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "ing.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}

	bedrock, err := embed.NewClient(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	emb := embed.NewTitanEmbedder(bedrock, embed.TitanModelID)
	a := NewAtomizer(d, llm.NewBedrockGenerator(bedrock), emb)
	return a, d, ts
}

func TestAtomizerIngestSampleProducesFiveFeaturesAndVectors(t *testing.T) {
	ctx := context.Background()
	fb := &fakeBedrock{}
	a, d, _ := setupAtomizer(t, fb)

	chunks, err := ParseFile("../../testdata/fsd-sample.md", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 5 {
		t.Fatalf("fixture should yield 5 chunks, got %d", len(chunks))
	}

	res, err := a.Ingest(ctx, chunks, "test-run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Features) != 5 {
		t.Errorf("expected 5 features in result, got %d", len(res.Features))
	}

	assertRowCount(t, d, "features", 5)
	assertRowCount(t, d, "feature_vec", 5)
	assertRowCount(t, d, "embedding_cache", 5)

	// Idempotency: re-run with same input ⇒ same counts, no embed call.
	pre := fb.embedCalls.Load()
	_, err = a.Ingest(ctx, chunks, "test-run-2")
	if err != nil {
		t.Fatal(err)
	}
	assertRowCount(t, d, "features", 5)
	assertRowCount(t, d, "feature_vec", 5)
	if fb.embedCalls.Load() != pre {
		t.Errorf("re-run made %d new embedding calls; expected 0",
			fb.embedCalls.Load()-pre)
	}
}

func TestAtomizerResumeSkipsCompletedFeatures(t *testing.T) {
	ctx := context.Background()
	fb := &fakeBedrock{}
	a, d, _ := setupAtomizer(t, fb)

	chunks, err := ParseFile("../../testdata/fsd-sample.md", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ingest(ctx, chunks, "resume-run"); err != nil {
		t.Fatal(err)
	}

	preAtomize := fb.atomizeCalls.Load()
	preEmbed := fb.embedCalls.Load()
	progress := make([]int, 0, len(chunks))
	resumeAtomizer := NewAtomizer(d, a.generator, a.embedder,
		WithResume(true),
		WithProgress(func(done, total int) {
			if total != len(chunks) {
				t.Errorf("progress total = %d, want %d", total, len(chunks))
			}
			progress = append(progress, done)
		}),
	)
	res, err := resumeAtomizer.Ingest(ctx, chunks, "resume-run")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Features) != len(chunks) {
		t.Fatalf("features = %d, want %d", len(res.Features), len(chunks))
	}
	if got := fb.atomizeCalls.Load(); got != preAtomize {
		t.Fatalf("resume atomized %d extra chunks", got-preAtomize)
	}
	if got := fb.embedCalls.Load(); got != preEmbed {
		t.Fatalf("resume embedded %d extra chunks", got-preEmbed)
	}
	if len(progress) != len(chunks) || progress[len(progress)-1] != len(chunks) {
		t.Fatalf("progress callbacks = %v, want final %d", progress, len(chunks))
	}
}

func TestAtomizerResumeSkipsCompletedChunks(t *testing.T) {
	ctx := context.Background()
	fb := &fakeBedrock{}
	a, d, ts := setupAtomizer(t, fb)

	chunks, err := ParseFile("../../testdata/fsd-sample.md", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Ingest(ctx, chunks[:2], "resume-run"); err != nil {
		t.Fatal(err)
	}
	preAtomize := fb.atomizeCalls.Load()

	bedrock, err := embed.NewClient(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	var progressDone []int
	resumeAtomizer := NewAtomizer(d,
		llm.NewBedrockGenerator(bedrock),
		embed.NewTitanEmbedder(bedrock, embed.TitanModelID),
		WithResume(true),
		WithProgress(func(done, _ int) {
			progressDone = append(progressDone, done)
		}),
	)
	res, err := resumeAtomizer.Ingest(ctx, chunks, "resume-run")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(res.Features), len(chunks); got != want {
		t.Fatalf("features = %d, want %d", got, want)
	}
	if got, want := fb.atomizeCalls.Load()-preAtomize, int64(len(chunks)-2); got != want {
		t.Fatalf("resume atomized %d chunks, want %d", got, want)
	}
	assertRowCount(t, d, "features", len(chunks))
	assertRowCount(t, d, "feature_vec", len(chunks))
	if len(progressDone) == 0 || progressDone[len(progressDone)-1] != len(chunks) {
		t.Fatalf("progress = %v, want final done %d", progressDone, len(chunks))
	}
}

func TestFeatureRowIDStableAndPositive(t *testing.T) {
	for _, id := range []string{"FR-001", "FR-001", "FR-999999"} {
		r := FeatureRowID(id)
		if r <= 0 {
			t.Errorf("FeatureRowID(%q) = %d (must be positive)", id, r)
		}
	}
	if FeatureRowID("FR-001") == FeatureRowID("FR-002") {
		t.Error("expected distinct rowids for distinct ids")
	}
	a := FeatureRowID("FR-001")
	b := FeatureRowID("FR-001")
	if a != b {
		t.Errorf("FeatureRowID must be deterministic: %d vs %d", a, b)
	}
}

func TestStripFenceRemovesMarkdownWrapper(t *testing.T) {
	in := "```json\n{\"id\":\"FR-1\"}\n```"
	out := stripFence(in)
	if out != `{"id":"FR-1"}` {
		t.Errorf("stripFence: got %q", out)
	}
	if stripFence("plain") != "plain" {
		t.Error("stripFence should be a no-op for unfenced input")
	}
}

func assertRowCount(t *testing.T, d *db.DB, table string, want int) {
	t.Helper()
	var n int
	err := d.SQL().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&n)
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if n != want {
		t.Errorf("table %s: got %d rows, want %d", table, n, want)
	}
}
