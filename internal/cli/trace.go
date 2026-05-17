package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/code"
	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/fsd"
	"github.com/cax/fsdtrace/internal/match"
	"github.com/cax/fsdtrace/internal/report"
)

func newTraceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Run an end-to-end trace from source input to report",
	}
	cmd.AddCommand(newTraceGithubCmd())
	return cmd
}

func newTraceGithubCmd() *cobra.Command {
	var o traceGithubOptions
	cmd := &cobra.Command{
		Use:   "github <url>",
		Short: "Download a GitHub repo and generate a traceability report",
		Long: `Downloads a public GitHub repository into a temporary directory, then runs:
init -> ingest fsd -> index code -> match -> report.

Example:
  fsdtrace trace github https://github.com/spring-projects/spring-petclinic \
    --fsd examples/petclinic/fsd.md --out ./trace --format html
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.fsdPath == "" {
				return errors.New("--fsd is required")
			}
			if _, err := parseGitHubRepoURL(args[0]); err != nil {
				return err
			}
			if o.resume {
				if opts.runID == "" {
					return errors.New("--resume requires --run-id")
				}
				if o.checkoutDir == "" {
					return errors.New("--resume requires --checkout-dir for trace github")
				}
			}
			if _, err := os.Stat(o.fsdPath); err != nil {
				return fmt.Errorf("stat fsd %s: %w", o.fsdPath, err)
			}

			ctx, cancel := signalCtx()
			defer cancel()
			cfg, err := loadAppConfig(opts.cfgPath, nil)
			if err != nil {
				return err
			}
			var provider string
			cfg, provider, err = resolveProvider(cfg, o.provider, cmd.Flags().Changed("provider"))
			if err != nil {
				return err
			}
			o.provider = provider
			o.embeddingModel = cfg.model(modelEmbedding, o.embeddingModel, cmd.Flags().Changed("embedding-model"))
			o.atomizerModel = cfg.model(modelAtomizer, o.atomizerModel, cmd.Flags().Changed("atomizer-model"))
			o.judgmentModel = cfg.model(modelJudgment, o.judgmentModel, cmd.Flags().Changed("judgment-model"))

			usingExisting := false
			if o.resume && o.checkoutDir != "" {
				if dest, err := filepath.Abs(o.checkoutDir); err == nil {
					if empty, err := dirMissingOrEmpty(dest); err == nil {
						usingExisting = !empty
					}
				}
			}
			repoDir, cleanup, err := prepareGitHubCheckout(ctx, args[0], o)
			if err != nil {
				return err
			}
			defer cleanup()
			checkoutMsg := "downloaded"
			if usingExisting {
				checkoutMsg = "using existing checkout"
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s to %s\n", checkoutMsg, args[0], repoDir); err != nil {
				return err
			}

			return runTracePipeline(ctx, cmd, cfg, repoDir, o)
		},
	}
	cmd.Flags().StringVar(&o.fsdPath, "fsd", "", "FSD file to validate against the downloaded repository")
	cmd.Flags().StringVar(&o.ref, "ref", "", "GitHub branch, tag, or commit to download (default: repo default branch)")
	cmd.Flags().StringVar(&o.checkoutDir, "checkout-dir", "", "directory to extract the downloaded repository into (default: temporary directory)")
	cmd.Flags().BoolVar(&o.keepRepo, "keep-repo", false, "keep the temporary downloaded repository after the run")
	cmd.Flags().StringVar(&o.format, "format", "html", "output format: md|csv|html|json")
	cmd.Flags().StringVar(&o.outDir, "out", "./trace/", "output directory")
	cmd.Flags().BoolVar(&o.includeCallGraph, "include-call-graph", false, "include SCIP relationship support artifacts in the report")
	cmd.Flags().BoolVar(&o.runScipJava, "run-scip-java", false, "shell out to scip-java in the downloaded repo before indexing")
	cmd.Flags().StringVar(&o.scipJavaBin, "scip-java-bin", "scip-java", "scip-java executable name")
	cmd.Flags().BoolVar(&o.resume, "resume", false, "resume a trace run by skipping completed FSD, code, and match work for --run-id")
	cmd.Flags().IntVar(&o.topK, "top-k", match.DefaultTopK, "vec0 KNN candidates to consider per FR")
	cmd.Flags().IntVar(&o.matchConcurrency, "match-concurrency", 1, "FRs to match in parallel")
	cmd.Flags().StringVar(&o.provider, "provider", "", "model provider: bedrock|openai (flag > env > config > default)")
	cmd.Flags().StringVar(&o.atomizerModel, "atomizer-model", "", "model id for atomization (flag > env > config > provider default)")
	cmd.Flags().StringVar(&o.judgmentModel, "judgment-model", "", "model id for judgment (flag > env > config > provider default)")
	cmd.Flags().StringVar(&o.embeddingModel, "embedding-model", "", "embedding model id (flag > env > config > provider default)")
	cmd.Flags().StringVar(&o.fsdCassette, "fsd-cassette", "", "use a recorded Bedrock cassette for FSD ingestion")
	cmd.Flags().StringVar(&o.indexCassette, "index-cassette", "", "use a recorded Bedrock cassette for code indexing")
	cmd.Flags().StringVar(&o.matchCassette, "match-cassette", "", "use a recorded Bedrock cassette for matching")
	return cmd
}

type traceGithubOptions struct {
	fsdPath          string
	ref              string
	checkoutDir      string
	keepRepo         bool
	format           string
	outDir           string
	includeCallGraph bool
	runScipJava      bool
	scipJavaBin      string
	resume           bool
	topK             int
	matchConcurrency int
	provider         string
	atomizerModel    string
	judgmentModel    string
	embeddingModel   string
	fsdCassette      string
	indexCassette    string
	matchCassette    string
}

func prepareGitHubCheckout(ctx context.Context, rawURL string, o traceGithubOptions) (string, func(), error) {
	dest, cleanup, err := checkoutDestination(o)
	if err != nil {
		return "", cleanup, err
	}
	if o.resume && o.checkoutDir != "" {
		empty, err := dirMissingOrEmpty(dest)
		if err != nil {
			return "", cleanup, err
		}
		if !empty {
			return dest, cleanup, nil
		}
	}
	if err := downloadGitHubRepo(ctx, rawURL, o.ref, dest); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return dest, cleanup, nil
}

func checkoutDestination(o traceGithubOptions) (string, func(), error) {
	cleanup := func() {}
	if o.checkoutDir == "" {
		dir, err := os.MkdirTemp("", "fsdtrace-github-*")
		if err != nil {
			return "", cleanup, fmt.Errorf("create temp checkout: %w", err)
		}
		if !o.keepRepo {
			cleanup = func() { _ = os.RemoveAll(dir) }
		}
		return dir, cleanup, nil
	}
	dest, err := filepath.Abs(o.checkoutDir)
	if err != nil {
		return "", cleanup, fmt.Errorf("resolve checkout dir: %w", err)
	}
	empty, err := dirMissingOrEmpty(dest)
	if err != nil {
		return "", cleanup, err
	}
	if !empty {
		if o.resume {
			return dest, cleanup, nil
		}
		return "", cleanup, fmt.Errorf("checkout dir %s already exists and is not empty", dest)
	}
	return dest, cleanup, nil
}

func dirMissingOrEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	switch {
	case err == nil:
		return len(entries) == 0, nil
	case os.IsNotExist(err):
		return true, nil
	default:
		return false, fmt.Errorf("read checkout dir %s: %w", path, err)
	}
}

func runTracePipeline(
	ctx context.Context,
	cmd *cobra.Command,
	cfg appConfig,
	repoDir string,
	o traceGithubOptions,
) error {
	d, err := db.Open(ctx, opts.dbPath)
	if err != nil {
		return fmt.Errorf("opening db: %w", err)
	}
	defer func() { _ = d.Close() }()
	if err := d.ApplySchema(ctx); err != nil {
		return err
	}

	runBase := opts.runID
	if runBase == "" {
		runBase = fmt.Sprintf("trace-%d", time.Now().Unix())
	}
	if err := traceIngestFSD(ctx, cmd.ErrOrStderr(), d, cfg, o, runBase+"-fsd"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ingested FSD (run %s)\n", runBase+"-fsd"); err != nil {
		return err
	}
	if err := traceIndexCode(ctx, cmd.ErrOrStderr(), d, cfg, repoDir, o, runBase+"-index"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "indexed code (run %s)\n", runBase+"-index"); err != nil {
		return err
	}
	matchRunID := runBase + "-match"
	if err := traceMatch(ctx, cmd.ErrOrStderr(), d, cfg, o, matchRunID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "matched features (run %s)\n", matchRunID); err != nil {
		return err
	}
	return traceReport(ctx, cmd, d, o, matchRunID)
}

func traceIngestFSD(ctx context.Context, errOut io.Writer, d *db.DB, cfg appConfig, o traceGithubOptions, runID string) error {
	chunks, err := fsd.ParseFile(o.fsdPath, fsd.DefaultAnchorPattern)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return fmt.Errorf("no anchors matched %q in %s", fsd.DefaultAnchorPattern, o.fsdPath)
	}
	generator, err := newGenerator(o.provider, o.fsdCassette, cfg)
	if err != nil {
		return err
	}
	emb, err := newEmbedder(o.provider, o.fsdCassette, cfg, o.embeddingModel, embed.PurposeDocument)
	if err != nil {
		return err
	}
	progress := newProgress(errOut, "ingest fsd")
	defer progress.Finish()
	atomizer := fsd.NewAtomizer(d, generator, emb,
		fsd.WithModel(o.atomizerModel),
		fsd.WithLogger(slog.Default()),
		fsd.WithResume(o.resume),
		fsd.WithProgress(progress.Advance))
	_, err = atomizer.Ingest(ctx, chunks, runID)
	return err
}

func traceIndexCode(
	ctx context.Context,
	errOut io.Writer,
	d *db.DB,
	cfg appConfig,
	repoDir string,
	o traceGithubOptions,
	runID string,
) error {
	emb, err := newEmbedder(o.provider, o.indexCassette, cfg, o.embeddingModel, embed.PurposeDocument)
	if err != nil {
		return err
	}
	progress := newProgress(errOut, "index code")
	defer progress.Finish()
	indexer := code.NewIndexer(d, emb,
		code.WithResume(o.resume),
		code.WithProgress(progress.Advance))

	scipPath := ""
	if o.runScipJava {
		scipPath = filepath.Join(repoDir, "index.scip")
		if err := code.RunScipJava(ctx, repoDir, scipPath, o.scipJavaBin); err != nil {
			if errors.Is(err, code.ErrScipJavaMissing) {
				return err
			}
			return fmt.Errorf("scip-java: %w", err)
		}
	}
	_, err = indexer.Index(ctx, repoDir, scipPath, runID)
	return err
}

func traceMatch(ctx context.Context, errOut io.Writer, d *db.DB, cfg appConfig, o traceGithubOptions, runID string) error {
	generator, err := newGenerator(o.provider, o.matchCassette, cfg)
	if err != nil {
		return err
	}
	emb, err := newEmbedder(o.provider, o.matchCassette, cfg, o.embeddingModel, embed.PurposeQuery)
	if err != nil {
		return err
	}
	pipeOpts := []match.PipelineOption{
		match.WithTopK(o.topK),
		match.WithMatchConcurrency(o.matchConcurrency),
		match.WithJudgmentModel(o.judgmentModel),
		match.WithResume(o.resume),
	}
	if o.provider == ProviderOpenAI {
		pipeOpts = append(pipeOpts,
			match.WithJudgmentMaxTokens(openAIJudgmentMaxTokens),
			match.WithJudgmentBatchSize(openAIJudgmentBatchSize),
		)
	}
	progress := newProgress(errOut, "match")
	defer progress.Finish()
	pipeOpts = append(pipeOpts, match.WithProgress(progress.Advance))
	pipe := match.NewPipeline(d, generator, emb, pipeOpts...)
	_, err = pipe.MatchAll(ctx, runID, nil)
	return err
}

func traceReport(ctx context.Context, cmd *cobra.Command, d *db.DB, o traceGithubOptions, runID string) error {
	rep, err := report.LoadWithOptions(ctx, d, runID, report.Options{
		IncludeCallGraph: o.includeCallGraph,
	})
	if err != nil {
		return err
	}
	switch o.format {
	case "md":
		err = report.WriteMarkdown(rep, o.outDir)
	case "csv":
		err = report.WriteCSV(rep, o.outDir)
	case "html":
		err = report.WriteHTML(rep, o.outDir)
	case "json":
		err = report.WriteJSON(rep, o.outDir)
	default:
		return fmt.Errorf("unknown format %q (want md|csv|html|json)", o.format)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(),
		"wrote %s report (run %s) to %s\n",
		o.format, fallback(rep.RunID, "<no matches>"), o.outDir)
	return err
}
