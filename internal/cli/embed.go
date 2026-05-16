package cli

import "github.com/spf13/cobra"

func newEmbedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Compute embeddings via Bedrock Titan and populate vec0 (Phase 2/3)",
		RunE: func(*cobra.Command, []string) error {
			return errNotImplemented
		},
	}
	cmd.Flags().String("what", "all", "what to embed: features|artifacts|all")
	return cmd
}
