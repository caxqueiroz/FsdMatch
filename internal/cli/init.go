package cli

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/db"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the SQLite database and apply the schema",
		Long:  "Creates --db if missing, applies schema.sql, and verifies the vec0 virtual tables are present. Idempotent.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			d, err := db.Open(ctx, opts.dbPath)
			if err != nil {
				return fmt.Errorf("opening %s: %w", opts.dbPath, err)
			}
			defer func() { _ = d.Close() }()

			if err := d.ApplySchema(ctx); err != nil {
				return fmt.Errorf("applying schema: %w", err)
			}

			slog.InfoContext(ctx, "initialised database",
				"path", opts.dbPath,
				"embedding_dim", db.EmbeddingDim)
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "initialised %s\n", opts.dbPath); err != nil {
				return err
			}
			return nil
		},
	}
}
