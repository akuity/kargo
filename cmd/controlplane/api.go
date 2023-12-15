package main

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
			scheme, err := newSchemeForAPI()
			if err != nil {
				return errors.Wrap(err, "new scheme for API")
			}
			internalClient, err := newClientForAPI(ctx, restCfg, scheme)
			if err != nil {
				return errors.Wrap(err, "create internal Kubernetes client")
			}
			kubeClientOptions := kubernetes.ClientOptions{
				NewInternalClient: func(context.Context, *rest.Config, *runtime.Scheme) (client.Client, error) {
					return internalClient, nil
				},
			}
			if cfg.OIDCConfig != nil {
				kubeClientOptions.GlobalServiceAccountNamespaces = cfg.OIDCConfig.GlobalServiceAccountNamespaces
			}
			kubeClient, err := kubernetes.NewClient(ctx, restCfg, kubeClientOptions)
			if err != nil {
				return errors.Wrap(err, "create Kubernetes client")
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

	// Index ServiceAccounts by ODIC email
	if err := kubeclient.IndexServiceAccountsByOIDCEmail(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index service accounts by oidc email")
	}
	// Index ServiceAccounts by OIDC groups
	if err := kubeclient.IndexServiceAccountsByOIDCGroups(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index service accounts by oidc groups")
	}
	// Index ServiceAccounts by OIDC subjects
	if err := kubeclient.IndexServiceAccountsByOIDCSubjects(ctx, mgr); err != nil {
		return nil, errors.Wrap(err, "index service accounts by oidc subjects")
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(errors.Wrap(err, "start manager"))
		}
	}()

	return mgr.GetClient(), nil
}

func newSchemeForAPI() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := kubescheme.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add Kubernetes api to scheme")
	}
	if err := kargoapi.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add kargo api to scheme")
	}
	return scheme, nil
}
