package cmd

import (
	"github.com/spf13/cobra"

	"github.com/akuityio/kargo/internal/api/proxy"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/logging"
)

func newAPIProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api-proxy",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg := config.NewAPIProxyConfig()
			logging.LoggerFromContext(ctx).Logger.SetLevel(cfg.LogLevel)
			srv := proxy.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
