package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/match"
)

func newMatchCmd() *cobra.Command {
	var (
		feature        string
		topK           int
		judgmentModel  string
		rejudgeModel   string
		rejudgeDrifts  bool
		embeddingModel string
		cassettePath   string
	)
	cmd := &cobra.Command{
		Use:   "match",
		Short: "Match FRs to code artifacts via anchor + KNN + Bedrock judgment",
		Long: `Runs the matcher pipeline over every feature row (or just --fr <id>) in the
database. For each FR: extracts deterministic anchors, retrieves up to --top-k
candidate artifacts via vec0 KNN merged with anchor matches, asks Claude to
judge each candidate, downgrades any verdict missing concrete evidence to
"unrelated" (SPEC §7.4 hard rule), then decorates each match with test
coverage from the tests table. Re-runs are idempotent within a run id.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			cfg, err := loadAppConfig(opts.cfgPath, nil)
			if err != nil {
				return err
			}
			embeddingModel := cfg.model(modelEmbedding, embeddingModel, cmd.Flags().Changed("embedding-model"))
			judgmentModel := cfg.model(modelJudgment, judgmentModel, cmd.Flags().Changed("judgment-model"))
			rejudgeModel := cfg.model(modelRejudge, rejudgeModel, cmd.Flags().Changed("rejudge-model"))

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
			pipe := match.NewPipeline(d, bedrock, emb,
				match.WithTopK(topK),
				match.WithJudgmentModel(judgmentModel),
			)

			runID := opts.runID
			if runID == "" {
				runID = fmt.Sprintf("match-%d", time.Now().Unix())
			}

			var only []string
			if feature != "" {
				only = []string{feature}
			}
			summary, err := pipe.MatchAll(ctx, runID, only)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"matched %d candidates (run %s): implements=%d drifts=%d unrelated=%d tested=%d\n",
				summary.TotalMatches, summary.RunID,
				summary.Implements, summary.Drifts, summary.Unrelated, summary.Tested); err != nil {
				return err
			}

			if rejudgeDrifts {
				rj, err := pipe.RejudgeDrifts(ctx, summary.RunID, rejudgeModel)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(),
					"rejudged %d drifts (model %s): promoted=%d still_drifts=%d downgraded=%d\n",
					rj.Total, rj.Model, rj.PromotedToImplements, rj.StillDrifts, rj.DowngradedToUnrelated)
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&feature, "fr", "", "match a single feature ID, e.g. FR-042")
	cmd.Flags().IntVar(&topK, "top-k", match.DefaultTopK, "vec0 KNN candidates to consider per FR")
	cmd.Flags().StringVar(&judgmentModel, "judgment-model", "", "Bedrock Claude model id for judgment (flag > env > config > default)")
	cmd.Flags().StringVar(&embeddingModel, "embedding-model", "", "Bedrock Titan model id for embeddings (flag > env > config > default)")
	cmd.Flags().StringVar(&cassettePath, "cassette", "", "use a recorded Bedrock cassette (skips live calls)")
	cmd.Flags().BoolVar(&rejudgeDrifts, "rejudge-drifts", false, "after the first pass, re-judge every drifts verdict with --rejudge-model (SPEC §7.4)")
	cmd.Flags().StringVar(&rejudgeModel, "rejudge-model", "", "stronger Bedrock Claude model used for the --rejudge-drifts second pass (flag > env > config > default)")
	return cmd
}
