package main

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
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
)

func Execute(ctx context.Context) error {
	rootCmd.AddCommand(newAPICommand())
	rootCmd.AddCommand(newControllerCommand())
	rootCmd.AddCommand(newExternalWebhooksServerCommand())
	rootCmd.AddCommand(newGarbageCollectorCommand())
	rootCmd.AddCommand(newKubernetesWebhooksServerCommand())
	rootCmd.AddCommand(newManagementControllerCommand())
	rootCmd.AddCommand(newVersionCommand())
	return rootCmd.ExecuteContext(ctx)
}
