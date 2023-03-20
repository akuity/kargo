package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:               "kargo",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Initialize context
			ctx := context.Background()
			cmd.SetContext(ctx)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
)

func Execute() error {
	rootCmd.AddCommand(newAPICommand())
	rootCmd.AddCommand(newAPIProxyCommand())
	rootCmd.AddCommand(newControllerCommand())
	rootCmd.AddCommand(newVersionCommand())
	return rootCmd.Execute()
}
