package db

import (
	"context"
	"database/sql"
	"math/rand"
	"path/filepath"
	"testing"
)

// TestVec0KNNSmoke is the Phase 1 acceptance smoke: insert 50 random
// 1024-dim vectors into artifact_vec, query top-5 against a known query
// vector, verify the result set is non-empty and sorted by distance
// ascending.
func TestVec0KNNSmoke(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "smoke.db")

	d, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if err := d.ApplySchema(ctx); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	// Verify all expected tables exist.
	mustHaveTables(t, d.SQL(), []string{
		"features", "code_artifacts", "relationships",
		"tests", "matches", "runs", "embedding_cache",
		"feature_vec", "artifact_vec",
	})

	// Insert 50 random 1024-d vectors via the writer goroutine.
	const n = 50
	r := rand.New(rand.NewSource(42))
	vectors := make([][]float32, n)
	for i := range vectors {
		v := make([]float32, EmbeddingDim)
		for j := range v {
			v[j] = r.Float32()
		}
		vectors[i] = v
	}

	err = d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO artifact_vec(rowid, embedding) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		defer func() { _ = stmt.Close() }()
		for i, v := range vectors {
			if _, err := stmt.ExecContext(ctx, i+1, PackFloat32(v)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("insert vectors: %v", err)
	}

	// Query top-5 nearest to vectors[0]. Expect rowid=1 distance 0
	// at the head, plus a sorted ascending distance sequence.
	query := PackFloat32(vectors[0])
	rows, err := d.SQL().QueryContext(ctx,
		`SELECT rowid, distance FROM artifact_vec
		 WHERE embedding MATCH ? AND k = 5
		 ORDER BY distance`, query)
	if err != nil {
		t.Fatalf("knn query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	type hit struct {
		rowid int
		dist  float64
	}
	var got []hit
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.rowid, &h.dist); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, h)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("expected 5 hits, got %d: %+v", len(got), got)
	}
	if got[0].rowid != 1 {
		t.Errorf("expected nearest to be rowid 1 (vectors[0]), got %d", got[0].rowid)
	}
	if got[0].dist > 1e-6 {
		t.Errorf("expected distance ≈ 0 for self, got %f", got[0].dist)
	}
	for i := 1; i < len(got); i++ {
		if got[i].dist < got[i-1].dist {
			t.Errorf("not sorted ascending at i=%d: %f < %f", i, got[i].dist, got[i-1].dist)
		}
	}
}

func TestPackUnpackFloat32(t *testing.T) {
	in := []float32{0, 1, -1, 3.14159, 1e-9, -1e9}
	b := PackFloat32(in)
	out, err := UnpackFloat32(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != len(out) {
		t.Fatalf("len mismatch: %d vs %d", len(in), len(out))
	}
	for i := range in {
		if in[i] != out[i] {
			t.Errorf("at %d: in=%g out=%g", i, in[i], out[i])
		}
	}
	if _, err := UnpackFloat32([]byte{1, 2, 3}); err == nil {
		t.Error("expected error for non-multiple-of-4 input")
	}
}

func mustHaveTables(t *testing.T, sqlDB *sql.DB, names []string) {
	t.Helper()
	ctx := context.Background()
	for _, name := range names {
		var got string
		err := sqlDB.QueryRowContext(ctx,
			`SELECT name FROM sqlite_master WHERE name = ? AND type IN ('table','view')`,
			name).Scan(&got)
		if err != nil {
			t.Errorf("table %q missing: %v", name, err)
		}
	}
}
