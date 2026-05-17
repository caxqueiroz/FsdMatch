package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
)

func newIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest source documents into the database",
	}
	cmd.AddCommand(newIngestFsdCmd())
	return cmd
}

func newIngestFsdCmd() *cobra.Command {
	var (
		anchorPattern  string
		atomizerModel  string
		embeddingModel string
		cassettePath   string
		providerFlag   string
		resume         bool
	)
	c := &cobra.Command{
		Use:   "fsd <path>",
		Short: "Atomize an FSD file into FR objects and embed them",
		Long: `Splits the FSD by FR anchor, calls the configured model to
extract structured Functional Requirements, embeds each FR,
and stores both in the SQLite database. Re-runs are idempotent: features
are upserted by id and feature_vec rows replaced.

Use --provider bedrock (default) with BEDROCK_BASE_URL, or --provider openai
with OPENAI_API_KEY. The --cassette flag substitutes a recorded Bedrock
responses file, used by tests and offline smoke runs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signalCtx()
			defer cancel()
			if resume && opts.runID == "" {
				return errors.New("--resume requires --run-id")
			}

			cfg, err := loadAppConfig(opts.cfgPath, nil)
			if err != nil {
				return err
			}
			var provider string
			cfg, provider, err = resolveProvider(cfg, providerFlag, cmd.Flags().Changed("provider"))
			if err != nil {
				return err
			}
			embeddingModel := cfg.model(modelEmbedding, embeddingModel, cmd.Flags().Changed("embedding-model"))
			atomizerModel := cfg.model(modelAtomizer, atomizerModel, cmd.Flags().Changed("atomizer-model"))

			path := args[0]
			chunks, err := fsd.ParseFile(path, anchorPattern)
			if err != nil {
				return err
			}
			if len(chunks) == 0 {
				return fmt.Errorf("no anchors matched %q in %s", anchorPattern, path)
			}

			d, err := db.Open(ctx, opts.dbPath)
			if err != nil {
				return fmt.Errorf("opening db: %w", err)
			}
			defer func() { _ = d.Close() }()
			if err := d.ApplySchema(ctx); err != nil {
				return err
			}

			generator, err := newGenerator(provider, cassettePath, cfg)
			if err != nil {
				return err
			}
			emb, err := newEmbedder(provider, cassettePath, cfg, embeddingModel, embed.PurposeDocument)
			if err != nil {
				return err
			}
			progress := newProgress(cmd.ErrOrStderr(), "ingest fsd")
			defer progress.Finish()
			atomizer := fsd.NewAtomizer(d, generator, emb,
				fsd.WithModel(atomizerModel),
				fsd.WithLogger(slog.Default()),
				fsd.WithResume(resume),
				fsd.WithProgress(progress.Advance))

			runID := opts.runID
			if runID == "" {
				runID = fmt.Sprintf("ingest-%d", time.Now().Unix())
			}
			res, err := atomizer.Ingest(ctx, chunks, runID)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"ingested %d features (run %s)\n", len(res.Features), res.RunID)
			return err
		},
	}
	c.Flags().StringVar(&anchorPattern, "anchor-pattern", fsd.DefaultAnchorPattern, "regex used to split chunks")
	c.Flags().StringVar(&providerFlag, "provider", "", "model provider: bedrock|openai (flag > env > config > default)")
	c.Flags().StringVar(&atomizerModel, "atomizer-model", "", "model id for atomization (flag > env > config > provider default)")
	c.Flags().StringVar(&embeddingModel, "embedding-model", "", "embedding model id (flag > env > config > provider default)")
	c.Flags().StringVar(&cassettePath, "cassette", "", "use a recorded Bedrock cassette (skips live calls)")
	c.Flags().BoolVar(&resume, "resume", false, "skip FSD chunks already completed for --run-id")
	return c
}
