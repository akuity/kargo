package cmd

import (
	"github.com/spf13/cobra"

	"github.com/akuityio/kargo/internal/api/server"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/logging"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg := config.NewAPIConfig()
			logging.LoggerFromContext(ctx).Logger.SetLevel(cfg.LogLevel)
			srv := server.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
