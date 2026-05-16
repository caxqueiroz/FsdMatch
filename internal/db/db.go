// Package db owns the SQLite connection, schema, and the single-writer
// goroutine that serialises mutations against modernc.org/sqlite.
//
// Hard constraint (CLAUDE.md): every connection runs WAL,
// busy_timeout=5000, foreign_keys=ON. The blank import of
// modernc.org/sqlite/vec auto-registers the vec0 extension.
package db

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	_ "modernc.org/sqlite"     // SQL driver
	_ "modernc.org/sqlite/vec" // registers vec0 via sqlite3_auto_extension
)

//go:embed schema.sql
var schemaSQL string

// EmbeddingDim is the fixed vec0 dimension. Matches Titan v2.
const EmbeddingDim = 1024

// DB wraps *sql.DB and an attached writer goroutine.
type DB struct {
	sql    *sql.DB
	writer *Writer
}

// Open opens (or creates) the SQLite file at path, applies the connection
// pragmas required by the spec, and starts the single-writer goroutine.
// Callers must Close the returned DB.
func Open(ctx context.Context, path string) (*DB, error) {
	dsn := buildDSN(path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite at %s: %w", path, err)
	}
	// Force a single, serialised connection for writes via the writer
	// goroutine; reads still go through the same pool but modernc/sqlite
	// is single-writer regardless.
	sqlDB.SetMaxOpenConns(0)

	if err := applyPragmas(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	d := &DB{sql: sqlDB}
	d.writer = newWriter(sqlDB)
	d.writer.start(ctx)
	return d, nil
}

// SQL returns the underlying *sql.DB for read queries.
// Writes must go through Writer().
func (d *DB) SQL() *sql.DB { return d.sql }

// Writer returns the single-writer handle.
func (d *DB) Writer() *Writer { return d.writer }

// Close stops the writer goroutine and closes the database.
func (d *DB) Close() error {
	if d.writer != nil {
		d.writer.stop()
	}
	return d.sql.Close()
}

// ApplySchema executes schema.sql. Idempotent: every CREATE uses IF NOT EXISTS.
func (d *DB) ApplySchema(ctx context.Context) error {
	if _, err := d.sql.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}
	return nil
}

// PackFloat32 packs a float32 slice into a little-endian byte buffer
// suitable for vec0 BLOB binding.
func PackFloat32(v []float32) []byte {
	out := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(f))
	}
	return out
}

// UnpackFloat32 reverses PackFloat32. Returns an error if len(b) % 4 != 0.
func UnpackFloat32(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("unpack: byte length %d not multiple of 4", len(b))
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}

func buildDSN(path string) string {
	// modernc.org/sqlite accepts pragmas in the DSN via _pragma=.
	// We still re-issue them in applyPragmas as defence in depth.
	params := []string{
		"_pragma=journal_mode(WAL)",
		"_pragma=busy_timeout(5000)",
		"_pragma=foreign_keys(ON)",
	}
	return path + "?" + strings.Join(params, "&")
}

func applyPragmas(ctx context.Context, sqlDB *sql.DB) error {
	stmts := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
	}
	for _, s := range stmts {
		if _, err := sqlDB.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("pragma %q: %w", s, err)
		}
	}
	return nil
}
