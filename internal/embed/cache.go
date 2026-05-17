package embed

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cax/fsdtrace/internal/db"
)

// ErrCacheMiss is returned by Cache.Get when the entry is absent.
var ErrCacheMiss = errors.New("embedding cache miss")

// CacheKey is the deterministic key for a (model config, text) pair.
// SHA-256 keeps it short and collision-resistant.
func CacheKey(model, text string) string {
	h := sha256.New()
	h.Write([]byte(model))
	h.Write([]byte{0x00})
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// Cache reads and writes the embedding_cache table.
type Cache struct {
	d *db.DB
}

// NewCache builds a cache backed by d.
func NewCache(d *db.DB) *Cache { return &Cache{d: d} }

// Get returns the cached vector for (model, text) or ErrCacheMiss.
func (c *Cache) Get(ctx context.Context, model, text string) ([]float32, error) {
	key := CacheKey(model, text)
	var blob []byte
	err := c.d.SQL().QueryRowContext(ctx,
		`SELECT embedding FROM embedding_cache WHERE key = ?`, key).Scan(&blob)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrCacheMiss
	case err != nil:
		return nil, fmt.Errorf("cache get: %w", err)
	}
	return db.UnpackFloat32(blob)
}

// Put stores v under (model, text). Idempotent: upserts by key.
func (c *Cache) Put(ctx context.Context, model, text string, v []float32) error {
	key := CacheKey(model, text)
	blob := db.PackFloat32(v)
	return c.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO embedding_cache(key, model, dim, embedding, created_at)
			 VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET
			   embedding = excluded.embedding,
			   model = excluded.model,
			   dim = excluded.dim,
			   created_at = excluded.created_at`,
			key, model, len(v), blob, time.Now().Unix())
		return err
	})
}

// Cached returns the embedding for text, consulting the cache first.
// On miss it calls the underlying Embedder and writes the result back.
// Safe for concurrent use; the underlying writer goroutine serialises
// writes.
func Cached(ctx context.Context, e Embedder, c *Cache, text string) ([]float32, error) {
	v, err := c.Get(ctx, e.Model(), text)
	switch {
	case err == nil:
		return v, nil
	case errors.Is(err, ErrCacheMiss):
		// fall through
	default:
		return nil, err
	}
	v, err = e.Embed(ctx, text)
	if err != nil {
		return nil, err
	}
	if err := c.Put(ctx, e.Model(), text, v); err != nil {
		return nil, fmt.Errorf("cache put: %w", err)
	}
	return v, nil
}
