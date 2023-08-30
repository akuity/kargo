package main

import (
	"fmt"
	"net"

	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo API Server")

			restCfg, err := kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
			if err != nil {
				return pkgerrors.Wrap(err, "error loading REST config")
			}
			client, err := kubernetes.NewClient(ctx, restCfg, kubernetes.ClientOptions{})
			if err != nil {
				return pkgerrors.Wrap(err, "error creating Kubernetes client")
			}

			cfg := config.ServerConfigFromEnv()

			if cfg.AdminConfig != nil {
				log.Info("admin account is enabled")
			}
			if cfg.OIDCConfig != nil {
				log.WithFields(log.Fields{
					"issuerURL":   cfg.OIDCConfig.IssuerURL,
					"clientID":    cfg.OIDCConfig.ClientID,
					"cliClientID": cfg.OIDCConfig.CLIClientID,
				}).Info("SSO via OpenID Connect is enabled")
			}

			srv, err := api.NewServer(cfg, client)
			if err != nil {
				return pkgerrors.Wrap(err, "error creating API server")
			}
			l, err := net.Listen(
				"tcp",
				fmt.Sprintf(
					"%s:%s",
					os.GetEnv("HOST", "0.0.0.0"),
					os.GetEnv("PORT", "8080"),
				),
			)
			if err != nil {
				return pkgerrors.Wrap(err, "error creating listener")
			}
			defer l.Close()

			return pkgerrors.Wrap(srv.Serve(ctx, l), "serve")
		},
	}
}
