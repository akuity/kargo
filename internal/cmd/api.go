package cmd

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/akuityio/kargo/internal/api/server"
	"github.com/akuityio/kargo/internal/config"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := signals.SetupSignalHandler()
			cfg := config.NewAPIConfig()
			srv := server.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
