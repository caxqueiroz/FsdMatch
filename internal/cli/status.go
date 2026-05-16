package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/db"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print a one-line summary of the database",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			d, err := db.Open(ctx, opts.dbPath)
			if err != nil {
				return fmt.Errorf("opening %s: %w", opts.dbPath, err)
			}
			defer func() { _ = d.Close() }()

			counts := map[string]int{}
			for _, t := range []string{"features", "code_artifacts", "matches", "tests", "runs"} {
				var n int
				row := d.SQL().QueryRowContext(ctx, "SELECT COUNT(*) FROM "+t)
				if err := row.Scan(&n); err != nil {
					return fmt.Errorf("counting %s: %w", t, err)
				}
				counts[t] = n
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"db=%s features=%d code_artifacts=%d matches=%d tests=%d runs=%d\n",
				opts.dbPath, counts["features"], counts["code_artifacts"],
				counts["matches"], counts["tests"], counts["runs"])
			return err
		},
	}
}
