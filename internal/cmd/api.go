package cmd

import (
	"github.com/spf13/cobra"

	"github.com/akuityio/kargo/internal/api/server"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/os"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := os.NotifyOnShutdown(cmd.Context())
			defer cancel()

			cfg := config.NewAPIConfig()
			srv := server.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
