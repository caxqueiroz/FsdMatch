package cli

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
)

func TestEmbedCommandPopulatesFeatureAndArtifactVectors(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "embed.db")
	seedEmbedCommandDB(t, ctx, dbPath)
	t.Setenv(EnvBedrockBaseURL, "https://bedrock.local")

	out, err := executeRoot(t, "--db", dbPath, "embed", "--what", "all")
	if err != nil {
		t.Fatalf("embed command: %v", err)
	}
	if !strings.Contains(out, "embedded 1 features, 1 artifacts") {
		t.Fatalf("unexpected output %q", out)
	}

	d, err := db.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	assertCLIEmbedCount(t, d, "feature_vec", 1)
	assertCLIEmbedCount(t, d, "artifact_vec", 1)
	assertCLIEmbedCount(t, d, "embedding_cache", 2)

	var artifactVecRows int
	err = d.SQL().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM artifact_vec_rowids WHERE rowid = 101`).Scan(&artifactVecRows)
	if err != nil {
		t.Fatal(err)
	}
	if artifactVecRows != 1 {
		t.Fatalf("artifact_vec rowid invariant broken: rowid 101 count = %d", artifactVecRows)
	}
}

func TestEmbedCommandRejectsInvalidWhatBeforeBedrock(t *testing.T) {
	_, err := executeRoot(t, "embed", "--what", "widgets")
	if err == nil {
		t.Fatal("expected invalid --what error")
	}
	if !strings.Contains(err.Error(), `unknown --what "widgets"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), EnvBedrockBaseURL) {
		t.Fatalf("invalid --what should fail before Bedrock setup: %v", err)
	}
}

func seedEmbedCommandDB(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	d, err := db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}

	err = d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO features
			  (id, title, description, acceptance, actor, inputs, outputs,
			   side_effects, non_functional, fsd_section, fsd_anchor, run_id, created_at)
			VALUES
			  ('FR-001', 'Create note', 'A user creates a note',
			   '["note is persisted"]', 'user', '[]', '[]', '[]', '[]',
			   'Notes', 'FR-001', 'seed-run', 1)
		`); err != nil {
			return fmt.Errorf("insert feature: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO code_artifacts
			  (id, kind, identifier, scip_symbol, package, class, method,
			   file, start_line, end_line, signature, annotations, source, run_id)
			VALUES
			  (101, 'rest_endpoint', 'POST /api/v1/notes', '',
			   'com.example', 'NoteController', 'createNote',
			   '/repo/NoteController.java', 10, 20,
			   'public Note createNote(Note note)', '{}',
			   'return service.create(note);', 'seed-run')
		`); err != nil {
			return fmt.Errorf("insert artifact: %w", err)
		}
		featureText := "Create note\nA user creates a note\n- note is persisted"
		if err := insertEmbeddingCache(ctx, tx, featureText, 1); err != nil {
			return err
		}
		artifactText := strings.Join([]string{
			"rest_endpoint: POST /api/v1/notes",
			"public Note createNote(Note note)",
			"return service.create(note);",
		}, "\n")
		if err := insertEmbeddingCache(ctx, tx, artifactText, 2); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func insertEmbeddingCache(ctx context.Context, tx *sql.Tx, text string, first float32) error {
	v := make([]float32, db.EmbeddingDim)
	v[0] = first
	_, err := tx.ExecContext(ctx, `
		INSERT INTO embedding_cache(key, model, dim, embedding, created_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		embed.CacheKey(embed.TitanModelID, text),
		embed.TitanModelID,
		db.EmbeddingDim,
		db.PackFloat32(v),
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert embedding cache: %w", err)
	}
	return nil
}

func executeRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	opts = globalOpts{}
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func assertCLIEmbedCount(t *testing.T, d *db.DB, table string, want int) {
	t.Helper()
	var got int
	err := d.SQL().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}
