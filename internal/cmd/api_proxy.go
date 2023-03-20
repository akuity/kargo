package cmd

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/akuityio/kargo/internal/api/proxy"
	"github.com/akuityio/kargo/internal/config"
)

func newAPIProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api-proxy",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := signals.SetupSignalHandler()
			cfg := config.NewAPIProxyConfig()
			srv := proxy.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
