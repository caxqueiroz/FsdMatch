package embed

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
)

type fakeEmbedder struct {
	model string
	calls atomic.Int64
	last  string
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	f.calls.Add(1)
	f.last = text
	v := make([]float32, db.EmbeddingDim)
	v[0] = float32(len(text))
	return v, nil
}

func (f *fakeEmbedder) Model() string { return f.model }

func TestCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}

	c := NewCache(d)
	if _, err := c.Get(ctx, "m", "x"); !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}

	v := make([]float32, db.EmbeddingDim)
	v[7] = 3.14
	if err := c.Put(ctx, "m", "x", v); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "m", "x")
	if err != nil {
		t.Fatal(err)
	}
	if got[7] != 3.14 {
		t.Errorf("round trip mismatch: %v", got[:10])
	}

	// Put again -> upsert, not duplicate.
	if err := c.Put(ctx, "m", "x", v); err != nil {
		t.Fatal(err)
	}
}

func TestEmbedCachedHitsCacheOnSecondCall(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}

	c := NewCache(d)
	emb := &fakeEmbedder{model: "fake"}

	for i := 0; i < 3; i++ {
		if _, err := Cached(ctx, emb, c, "same input"); err != nil {
			t.Fatal(err)
		}
	}
	if emb.calls.Load() != 1 {
		t.Errorf("expected exactly one upstream call, got %d", emb.calls.Load())
	}
}
