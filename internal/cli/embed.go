package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/code"
	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
)

func newEmbedCmd() *cobra.Command {
	var (
		what           string
		embeddingModel string
		cassettePath   string
	)
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Compute embeddings via Bedrock Titan and populate vec0 (Phase 2/3)",
		Long: `Recomputes vec0 rows for existing features and/or code artifacts.
This is useful after importing rows, changing the embedding model, or
repairing a database whose feature_vec/artifact_vec tables are incomplete.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			what, err := normalizeEmbedWhat(what)
			if err != nil {
				return err
			}

			ctx, cancel := signalCtx()
			defer cancel()

			cfg, err := loadAppConfig(opts.cfgPath, nil)
			if err != nil {
				return err
			}
			embeddingModel := cfg.model(modelEmbedding, embeddingModel, cmd.Flags().Changed("embedding-model"))

			d, err := db.Open(ctx, opts.dbPath)
			if err != nil {
				return fmt.Errorf("opening db: %w", err)
			}
			defer func() { _ = d.Close() }()
			if err := d.ApplySchema(ctx); err != nil {
				return err
			}

			bedrock, err := newBedrockClient(cassettePath, cfg)
			if err != nil {
				return err
			}
			emb := embed.NewTitanEmbedder(bedrock, embeddingModel)
			summary, err := embedExistingRows(ctx, d, emb, what)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"embedded %d features, %d artifacts\n",
				summary.FeatureCount, summary.ArtifactCount)
			return err
		},
	}
	cmd.Flags().StringVar(&what, "what", "all", "what to embed: features|artifacts|all")
	cmd.Flags().StringVar(&embeddingModel, "embedding-model", "", "Bedrock Titan model id for embeddings (flag > env > config > default)")
	cmd.Flags().StringVar(&cassettePath, "cassette", "", "use a recorded Bedrock cassette (skips live calls)")
	return cmd
}

type embedSummary struct {
	FeatureCount  int
	ArtifactCount int
}

func normalizeEmbedWhat(what string) (string, error) {
	what = strings.TrimSpace(strings.ToLower(what))
	switch what {
	case "", "all":
		return "all", nil
	case "features", "artifacts":
		return what, nil
	default:
		return "", fmt.Errorf("unknown --what %q (want features|artifacts|all)", what)
	}
}

func embedExistingRows(ctx context.Context, d *db.DB, emb embed.Embedder, what string) (*embedSummary, error) {
	cache := embed.NewCache(d)
	summary := &embedSummary{}
	if what == "all" || what == "features" {
		n, err := embedFeatureRows(ctx, d, emb, cache)
		if err != nil {
			return nil, err
		}
		summary.FeatureCount = n
	}
	if what == "all" || what == "artifacts" {
		n, err := embedArtifactRows(ctx, d, emb, cache)
		if err != nil {
			return nil, err
		}
		summary.ArtifactCount = n
	}
	return summary, nil
}

func embedFeatureRows(ctx context.Context, d *db.DB, emb embed.Embedder, cache *embed.Cache) (int, error) {
	rows, err := d.SQL().QueryContext(ctx, `
		SELECT id, title, description, acceptance
		FROM features
		ORDER BY id
	`)
	if err != nil {
		return 0, fmt.Errorf("query features: %w", err)
	}
	defer func() { _ = rows.Close() }()

	count := 0
	for rows.Next() {
		var (
			id          string
			title       string
			description string
			acceptance  string
		)
		if err := rows.Scan(&id, &title, &description, &acceptance); err != nil {
			return count, fmt.Errorf("scan feature: %w", err)
		}
		var criteria []string
		if err := json.Unmarshal([]byte(acceptance), &criteria); err != nil {
			return count, fmt.Errorf("decode feature %s acceptance: %w", id, err)
		}
		f := fsd.Feature{
			ID:          id,
			Title:       title,
			Description: description,
			Acceptance:  criteria,
		}
		v, err := embed.Cached(ctx, emb, cache, f.EmbeddingText())
		if err != nil {
			return count, fmt.Errorf("embed feature %s: %w", id, err)
		}
		if err := writeVecRow(ctx, d, "feature_vec", fsd.FeatureRowID(id), v); err != nil {
			return count, fmt.Errorf("write feature %s vector: %w", id, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("iterate features: %w", err)
	}
	return count, nil
}

func embedArtifactRows(ctx context.Context, d *db.DB, emb embed.Embedder, cache *embed.Cache) (int, error) {
	rows, err := d.SQL().QueryContext(ctx, `
		SELECT id, kind, identifier, signature, source
		FROM code_artifacts
		ORDER BY id
	`)
	if err != nil {
		return 0, fmt.Errorf("query artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	count := 0
	for rows.Next() {
		var (
			id         int64
			kind       string
			identifier string
			signature  sql.NullString
			source     sql.NullString
		)
		if err := rows.Scan(&id, &kind, &identifier, &signature, &source); err != nil {
			return count, fmt.Errorf("scan artifact: %w", err)
		}
		a := code.Artifact{
			Kind:       kind,
			Identifier: identifier,
			Signature:  nullString(signature),
			Source:     nullString(source),
		}
		v, err := embed.Cached(ctx, emb, cache, code.EmbeddingText(a))
		if err != nil {
			return count, fmt.Errorf("embed artifact %s: %w", identifier, err)
		}
		if err := writeVecRow(ctx, d, "artifact_vec", id, v); err != nil {
			return count, fmt.Errorf("write artifact %s vector: %w", identifier, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("iterate artifacts: %w", err)
	}
	return count, nil
}

func writeVecRow(ctx context.Context, d *db.DB, table string, rowID int64, v []float32) error {
	var deleteSQL, insertSQL string
	switch table {
	case "feature_vec":
		deleteSQL = `DELETE FROM feature_vec WHERE rowid = ?`
		insertSQL = `INSERT INTO feature_vec(rowid, embedding) VALUES (?, ?)`
	case "artifact_vec":
		deleteSQL = `DELETE FROM artifact_vec WHERE rowid = ?`
		insertSQL = `INSERT INTO artifact_vec(rowid, embedding) VALUES (?, ?)`
	default:
		return fmt.Errorf("unknown vec table %q", table)
	}
	blob := db.PackFloat32(v)
	return d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, deleteSQL, rowID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, insertSQL, rowID, blob)
		return err
	})
}

func nullString(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}
