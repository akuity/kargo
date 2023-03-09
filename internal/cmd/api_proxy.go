package cmd

import (
	"github.com/spf13/cobra"

	"github.com/akuityio/kargo/internal/api/proxy"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/os"
)

func newAPIProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api-proxy",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := os.NotifyOnShutdown(cmd.Context())
			defer cancel()

			cfg := config.NewAPIProxyConfig()
			srv := proxy.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
