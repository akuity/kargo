package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/api/server"
	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/logging"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.NewAPIConfig()
			logger := log.New()
			logger.SetLevel(cfg.LogLevel)
			ctx := logging.ContextWithLogger(cmd.Context(), logger.WithFields(nil))
			srv := server.NewServer(cfg)
			return srv.Serve(ctx)
		},
	}
}
