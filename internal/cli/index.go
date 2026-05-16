package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/code"
	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
)

func newIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index a source repository",
	}
	cmd.AddCommand(newIndexCodeCmd())
	return cmd
}

func newIndexCodeCmd() *cobra.Command {
	var (
		scipIndexPath  string
		runScipJava    bool
		scipJavaBin    string
		embeddingModel string
		cassettePath   string
	)
	c := &cobra.Command{
		Use:   "code <repo>",
		Short: "Index a Spring Boot repo: harvest annotations, optionally merge SCIP",
		Long: `Walks <repo> for .java files, harvests Spring annotations and JUnit tests
into code_artifacts and tests, and embeds each artifact via Titan v2.

If --scip-index points at an existing index.scip the SCIP layer fills
scip_symbol and inserts call-graph relationships. If --run-scip-java is
set, scip-java runs first inside <repo> to produce that file. Either
way the indexer succeeds when the harvester does.

Bedrock embedding access is via BEDROCK_BASE_URL or --cassette.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			cfg, err := loadAppConfig(opts.cfgPath, nil)
			if err != nil {
				return err
			}
			embeddingModel := cfg.model(modelEmbedding, embeddingModel, cmd.Flags().Changed("embedding-model"))

			repoRoot, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			info, err := os.Stat(repoRoot)
			if err != nil {
				return fmt.Errorf("stat %s: %w", repoRoot, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", repoRoot)
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
			emb := embed.NewTitanEmbedder(bedrock, embeddingModel)
			indexer := code.NewIndexer(d, emb)

			scipPath := scipIndexPath
			if runScipJava {
				out := filepath.Join(repoRoot, "index.scip")
				if err := code.RunScipJava(ctx, repoRoot, out, scipJavaBin); err != nil {
					if errors.Is(err, code.ErrScipJavaMissing) {
						return err
					}
					return fmt.Errorf("scip-java: %w", err)
				}
				scipPath = out
			}

			runID := opts.runID
			if runID == "" {
				runID = fmt.Sprintf("index-%d", time.Now().Unix())
			}
			res, err := indexer.Index(ctx, repoRoot, scipPath, runID)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"indexed %d artifacts, %d tests, %d relations (run %s, scip-merged %d)\n",
				res.ArtifactCount, res.TestCount, res.RelationCount, res.RunID, res.ScipMerged)
			return err
		},
	}
	c.Flags().StringVar(&scipIndexPath, "scip-index", "", "path to a pre-existing index.scip (skips scip-java)")
	c.Flags().BoolVar(&runScipJava, "run-scip-java", false, "shell out to scip-java in the repo first")
	c.Flags().StringVar(&scipJavaBin, "scip-java-bin", "scip-java", "scip-java executable name")
	c.Flags().StringVar(&embeddingModel, "embedding-model", "", "Bedrock Titan model id for embeddings (flag > env > config > default)")
	c.Flags().StringVar(&cassettePath, "cassette", "", "use a recorded Bedrock cassette (skips live calls)")
	return c
}
