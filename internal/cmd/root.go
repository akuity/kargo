package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/akuityio/kargo/internal/logging"
)

var (
	rootCmd = &cobra.Command{
		Use:               "kargo",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Initialize context
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			ctx = logging.ContextWithLogger(ctx, logging.LoggerFromContext(ctx))
			cmd.SetContext(ctx)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
)

func Execute(ctx context.Context) error {
	rootCmd.AddCommand(newAPICommand())
	rootCmd.AddCommand(newAPIProxyCommand())
	rootCmd.AddCommand(newControllerCommand())
	rootCmd.AddCommand(newVersionCommand())
	return rootCmd.ExecuteContext(ctx)
}
