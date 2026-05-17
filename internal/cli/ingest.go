package cli

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
)

// EnvBedrockBaseURL is the env variable read for the Bedrock route. The
// CLI never hardcodes the AWS endpoint (CLAUDE.md hard constraint).
const EnvBedrockBaseURL = "BEDROCK_BASE_URL"

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
	)
	c := &cobra.Command{
		Use:   "fsd <path>",
		Short: "Atomize an FSD file into FR objects and embed them",
		Long: `Splits the FSD by anchor (default FR-\d+), calls Claude on Bedrock to
extract structured Functional Requirements, embeds each FR via Bedrock,
and stores both in the SQLite database. Re-runs are idempotent: features
are upserted by id and feature_vec rows replaced.

Bedrock access is via the BEDROCK_BASE_URL env variable (a KrakenD route).
The --cassette flag substitutes a recorded responses file, used by tests
and offline smoke runs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			cfg, err := loadAppConfig(opts.cfgPath, nil)
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

			bedrock, err := newBedrockClient(cassettePath, cfg)
			if err != nil {
				return err
			}

			emb := embed.NewBedrockEmbedder(bedrock, embeddingModel, embed.PurposeDocument)
			atomizer := fsd.NewAtomizer(d, bedrock, emb,
				fsd.WithModel(atomizerModel),
				fsd.WithLogger(slog.Default()))

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
	c.Flags().StringVar(&atomizerModel, "atomizer-model", "", "Bedrock Claude model id for atomization (flag > env > config > default)")
	c.Flags().StringVar(&embeddingModel, "embedding-model", "", "Bedrock embedding model id (flag > env > config > default)")
	c.Flags().StringVar(&cassettePath, "cassette", "", "use a recorded Bedrock cassette (skips live calls)")
	return c
}

// newBedrockClient builds a BedrockClient honouring the env variable
// and an optional cassette file.
func newBedrockClient(cassettePath string, cfg appConfig) (*embed.BedrockClient, error) {
	if cassettePath != "" {
		cas, err := embed.LoadCassette(cassettePath)
		if err != nil {
			return nil, err
		}
		// Cassette plays back URL-independently; any non-empty base is fine.
		return embed.NewClient("https://cassette.local",
			embed.WithHTTPClient(cas.HTTPClient()))
	}
	base := cfg.bedrockURL()
	if base == "" {
		return nil, fmt.Errorf("%s is unset; set it to the KrakenD route or pass --cassette",
			EnvBedrockBaseURL)
	}
	return embed.NewClient(base,
		embed.WithHTTPClient(&http.Client{Timeout: 90 * time.Second}))
}
