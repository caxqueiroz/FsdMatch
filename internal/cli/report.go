package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/report"
)

func newReportCmd() *cobra.Command {
	var (
		format           string
		outDir           string
		runID            string
		includeCallGraph bool
	)
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate the traceability matrix in md|csv|html|json",
		Long: `Builds a Report from the most recent match run (or --run-id) and writes
the chosen format into --out:

  md   → coverage.md, drift.md, orphans.md
  csv  → matches.csv, drift.csv, orphans.csv
  html → index.html (single self-contained page)
  json → report.json (full lossless dump)
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			d, err := db.Open(ctx, opts.dbPath)
			if err != nil {
				return fmt.Errorf("opening db: %w", err)
			}
			defer func() { _ = d.Close() }()
			if err := d.ApplySchema(ctx); err != nil {
				return err
			}

			rep, err := report.LoadWithOptions(ctx, d, runID, report.Options{
				IncludeCallGraph: includeCallGraph,
			})
			if err != nil {
				return err
			}

			switch format {
			case "md":
				err = report.WriteMarkdown(rep, outDir)
			case "csv":
				err = report.WriteCSV(rep, outDir)
			case "html":
				err = report.WriteHTML(rep, outDir)
			case "json":
				err = report.WriteJSON(rep, outDir)
			default:
				return fmt.Errorf("unknown format %q (want md|csv|html|json)", format)
			}
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"wrote %s report (run %s) to %s\n",
				format, fallback(rep.RunID, "<no matches>"), outDir)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "md", "output format: md|csv|html|json")
	cmd.Flags().StringVar(&outDir, "out", "./trace/", "output directory")
	cmd.Flags().StringVar(&runID, "run-id-of-matches", "", "match run to report on (default: latest)")
	cmd.Flags().BoolVar(&includeCallGraph, "include-call-graph", false, "include SCIP relationship support artifacts in the report")
	return cmd
}

func fallback(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}
