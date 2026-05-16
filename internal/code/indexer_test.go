package code

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
)

type fakeEmbedder struct {
	calls atomic.Int64
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	f.calls.Add(1)
	v := make([]float32, db.EmbeddingDim)
	v[0] = float32(len(text))
	return v, nil
}

func (f *fakeEmbedder) Model() string { return "fake-embedder" }

func TestIndexerOnFixturePopulatesAllRequiredTables(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "ix.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}

	repoRoot, err := filepath.Abs("../../testdata/sample-spring-app")
	if err != nil {
		t.Fatal(err)
	}
	indexer := NewIndexer(d, &fakeEmbedder{})
	res, err := indexer.Index(ctx, repoRoot, "", "test-run")
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if res.ArtifactCount == 0 {
		t.Fatal("no artifacts written")
	}
	if res.TestCount == 0 {
		t.Fatal("no tests written")
	}

	// Distinct kinds must include each Phase 3 acceptance kind.
	kinds := mustQueryStringSet(t, d, "SELECT DISTINCT kind FROM code_artifacts")
	for _, want := range []string{
		"rest_endpoint",
		"kafka_listener",
		"scheduled_job",
		"security_rule",
		"entity",
		"config_props",
		"exception_handler",
	} {
		if _, ok := kinds[want]; !ok {
			t.Errorf("kind %q missing from code_artifacts; have %v", want, kinds)
		}
	}

	// artifact_vec count matches code_artifacts (via shadow rowid table).
	var artifacts, vecRows int
	err = d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM code_artifacts").Scan(&artifacts)
	if err != nil {
		t.Fatal(err)
	}
	err = d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_vec_rowids").Scan(&vecRows)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts != vecRows {
		t.Errorf("artifact_vec rows = %d; expected %d (one per artifact)", vecRows, artifacts)
	}

	// At least one test row, and it should link to a rest_endpoint artifact.
	var linked int
	err = d.SQL().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tests WHERE target_artifact IS NOT NULL").Scan(&linked)
	if err != nil {
		t.Fatal(err)
	}
	if linked == 0 {
		t.Error("expected at least one test linked to its target rest_endpoint")
	}

	// Re-run is idempotent: same counts, no orphan rows.
	res2, err := indexer.Index(ctx, repoRoot, "", "test-run-2")
	if err != nil {
		t.Fatal(err)
	}
	if res2.ArtifactCount != res.ArtifactCount {
		t.Errorf("re-run artifact count drifted: %d → %d", res.ArtifactCount, res2.ArtifactCount)
	}
	var artifacts2 int
	err = d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM code_artifacts").Scan(&artifacts2)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts2 != artifacts {
		t.Errorf("re-run left %d artifacts (was %d) — purge isn't idempotent", artifacts2, artifacts)
	}
}

func TestNormalizeMockPathReplacesDigitSegments(t *testing.T) {
	cases := map[string]string{
		"GET /api/v1/notes/1":     "GET /api/v1/notes/{id}",
		"DELETE /api/v1/notes/42": "DELETE /api/v1/notes/{id}",
		"POST /api/v1/notes":      "POST /api/v1/notes",
	}
	for in, want := range cases {
		got := normalizeMockPath(in)
		if got != want {
			t.Errorf("normalizeMockPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func mustQueryStringSet(t *testing.T, d *db.DB, query string) map[string]struct{} {
	t.Helper()
	rows, err := d.SQL().QueryContext(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]struct{}{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out[s] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
