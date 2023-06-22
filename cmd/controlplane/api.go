package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/api"
	libConfig "github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/kubeclient"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := libConfig.NewAPIConfig()
			rc, err := config.GetConfig()
			if err != nil {
				return errors.Wrap(err, "load kubeconfig")
			}
			rc.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
				return kubeclient.NewCredentialInjector(rt)
			}
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
			return srv.Serve(cmd.Context(), l)
		},
	}
}
