package cmd

import (
	log "github.com/sirupsen/logrus"
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
			cfg := config.NewAPIProxyConfig()
			logger := log.New()
			logger.SetLevel(cfg.LogLevel)
			ctx := logging.ContextWithLogger(cmd.Context(), logger.WithFields(nil))
			srv := proxy.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
