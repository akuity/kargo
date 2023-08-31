package main

import (
	"context"
	"fmt"
	"net"

	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/kubeclient"
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
			kubeClient, err := kubernetes.NewClient(ctx, restCfg, kubernetes.ClientOptions{
				NewInternalClient: newClientForAPI,
			})
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

			srv, err := api.NewServer(cfg, kubeClient)
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

func newClientForAPI(ctx context.Context, r *rest.Config, scheme *runtime.Scheme) (client.Client, error) {
	mgr, err := ctrl.NewManager(r, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "new manager")
	}
	// Index Promotions by Stage
	if err := kubeclient.IndexPromotionsByStage(ctx, mgr); err != nil {
		return nil, pkgerrors.Wrap(err, "index promotions by stage")
	}
	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(pkgerrors.Wrap(err, "start manager"))
		}
	}()
	return mgr.GetClient(), nil
}
