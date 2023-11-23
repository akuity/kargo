package main

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
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

			cfg := config.ServerConfigFromEnv()
			restCfg, err := kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
			if err != nil {
				return errors.Wrap(err, "error loading REST config")
			}
			kubeClient, err := kubernetes.NewClient(ctx, restCfg, kubernetes.ClientOptions{
				KargoNamespace:    cfg.KargoNamespace,
				NewInternalClient: newClientForAPI,
			})
			if err != nil {
				return errors.Wrap(err, "create Kubernetes client")
			}
			internalClient, err := newClientForAPI(ctx, restCfg, kubeClient.Scheme())
			if err != nil {
				return errors.Wrap(err, "create internal Kubernetes client")
			}

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

			srv := api.NewServer(cfg, kubeClient, internalClient)
			l, err := net.Listen(
				"tcp",
				fmt.Sprintf(
					"%s:%s",
					os.GetEnv("HOST", "0.0.0.0"),
					os.GetEnv("PORT", "8080"),
				),
			)
			if err != nil {
				return errors.Wrap(err, "error creating listener")
			}
			defer l.Close()

			return errors.Wrap(srv.Serve(ctx, l), "serve")
		},
	}
}

func newClientForAPI(ctx context.Context, r *rest.Config, scheme *runtime.Scheme) (client.Client, error) {
	mgr, err := ctrl.NewManager(r, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return nil, errors.Wrap(err, "new manager")
	}

	// Index Promotions by Stage
	if err := kubeclient.IndexPromotionsByStage(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index promotions by stage")
	}

	// Index Freight by Warehouse
	if err := kubeclient.IndexFreightByWarehouse(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index freight by warehouse")
	}

	// Index Freight by Stages in which it has been verified
	if err := kubeclient.IndexFreightByVerifiedStages(ctx, mgr); err != nil {
		return nil,
			errors.Wrap(err, "index Freight by Stages in which it has been verified")
	}

	// Index Freight by Stages for which it is approved
	if err :=
		kubeclient.IndexFreightByApprovedStages(ctx, mgr); err != nil {
		return nil,
			errors.Wrap(err, "index Freight by Stages for which it has been approved")
	}

	// Index ServiceAccounts by RBAC Groups
	if err := kubeclient.IndexServiceAccountsByRBACGroups(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index service accounts by rbac groups")
	}
	// Index ServiceAccounts by RBAC Subjects
	if err := kubeclient.IndexServiceAccountsByRBACSubjects(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index servi ce accounts by rbac subjects")
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(errors.Wrap(err, "start manager"))
		}
	}()

	return mgr.GetClient(), nil
}
