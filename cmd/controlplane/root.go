package main

import (
	"context"

	"github.com/spf13/cobra"

	controllermetrics "github.com/akuity/kargo/pkg/controller/metrics"
	"github.com/akuity/kargo/pkg/metrics"
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
	metricsCfg, err := metrics.ConfigFromEnv()
	if err != nil {
		return err
	}
	if metricsCfg.Enabled(metrics.FamilyRESTClient) {
		metrics.EnableRESTClientMetrics()
	}
	if !metricsCfg.Enabled(metrics.FamilyLifecycle) {
		controllermetrics.DisableLifecycleMetrics()
	}

	rootCmd.AddCommand(newAPICommand())
	rootCmd.AddCommand(newControllerCommand())
	rootCmd.AddCommand(newExternalWebhooksServerCommand())
	rootCmd.AddCommand(newGarbageCollectorCommand())
	rootCmd.AddCommand(newKubernetesWebhooksServerCommand())
	rootCmd.AddCommand(newManagementControllerCommand())
	rootCmd.AddCommand(newMigrateCommand())
	rootCmd.AddCommand(newVersionCommand())
	return rootCmd.ExecuteContext(ctx)
}
