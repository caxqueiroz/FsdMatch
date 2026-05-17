package code

import (
	"context"
	"errors"
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

type failingEmbedder struct {
	calls     atomic.Int64
	failAfter int64
}

func (f *failingEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	call := f.calls.Add(1)
	if f.failAfter > 0 && call > f.failAfter {
		return nil, errors.New("embed failed")
	}
	v := make([]float32, db.EmbeddingDim)
	v[0] = float32(len(text))
	return v, nil
}

func (f *failingEmbedder) Model() string { return "failing-embedder" }

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

func TestIndexerResumeSkipsExistingVectorsAndReusesArtifacts(t *testing.T) {
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
	broken := &failingEmbedder{failAfter: 1}
	if _, err := NewIndexer(d, broken).Index(ctx, repoRoot, "", "resume-index"); err == nil {
		t.Fatal("first index unexpectedly succeeded")
	}
	var artifactsBefore, vectorsBefore int
	if err := d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM code_artifacts").Scan(&artifactsBefore); err != nil {
		t.Fatal(err)
	}
	if err := d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_vec_rowids").Scan(&vectorsBefore); err != nil {
		t.Fatal(err)
	}
	if artifactsBefore == 0 || vectorsBefore != 1 {
		t.Fatalf("partial run artifacts=%d vectors=%d, want artifacts>0 and vectors=1", artifactsBefore, vectorsBefore)
	}

	embedder := &fakeEmbedder{}
	progress := make([]int, 0, artifactsBefore)
	res, err := NewIndexer(d, embedder,
		WithResume(true),
		WithProgress(func(done, total int) {
			if total != artifactsBefore {
				t.Errorf("progress total = %d, want %d", total, artifactsBefore)
			}
			progress = append(progress, done)
		}),
	).Index(ctx, repoRoot, "", "resume-index")
	if err != nil {
		t.Fatal(err)
	}
	if res.ArtifactCount != artifactsBefore {
		t.Fatalf("artifact count = %d, want %d", res.ArtifactCount, artifactsBefore)
	}
	if got := embedder.calls.Load(); got != int64(artifactsBefore-vectorsBefore) {
		t.Fatalf("resume embed calls = %d, want %d", got, artifactsBefore-vectorsBefore)
	}
	var artifactsAfter, vectorsAfter int
	if err := d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM code_artifacts").Scan(&artifactsAfter); err != nil {
		t.Fatal(err)
	}
	if err := d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_vec_rowids").Scan(&vectorsAfter); err != nil {
		t.Fatal(err)
	}
	if artifactsAfter != artifactsBefore || vectorsAfter != artifactsBefore {
		t.Fatalf("after resume artifacts=%d vectors=%d, want both %d", artifactsAfter, vectorsAfter, artifactsBefore)
	}
	if len(progress) != artifactsBefore || progress[len(progress)-1] != artifactsBefore {
		t.Fatalf("progress callbacks = %v, want final %d", progress, artifactsBefore)
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
