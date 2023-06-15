package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/kubeclient"
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
			rc, err := cfg.RESTConfig()
			if err != nil {
				return errors.Wrap(err, "load kubeconfig")
			}
			rc.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
				return kubeclient.NewCredentialInjector(rt)
			}
			logger := log.New()
			logger.SetLevel(cfg.LogLevel)
			ctx := logging.ContextWithLogger(cmd.Context(), logger.WithFields(nil))
			srv, err := api.NewServer(cfg, rc)
			if err != nil {
				return errors.Wrap(err, "new api server")
			}
			l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
			if err != nil {
				return errors.Wrap(err, "new listener")
			}
			defer func() {
				_ = l.Close()
			}()
			return srv.Serve(ctx, l)
		},
	}
}
