package main

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/types"
	versionpkg "github.com/akuity/kargo/internal/version"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo API Server")

			cfg := config.ServerConfigFromEnv()
			restCfg, err := kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
			if err != nil {
				return fmt.Errorf("error loading REST config: %w", err)
			}

			scheme := runtime.NewScheme()
			if err = kubescheme.AddToScheme(scheme); err != nil {
				return fmt.Errorf("add Kubernetes api to scheme: %w", err)
			}

			var rolloutsEnabled bool
			if types.MustParseBool(os.GetEnv("ROLLOUTS_INTEGRATION_ENABLED", "true")) {
				if argoRolloutsExists(ctx, restCfg) {
					log.Info("Argo Rollouts integration is enabled")
					if err = rollouts.AddToScheme(scheme); err != nil {
						return fmt.Errorf("add argo rollouts api to scheme: %w", err)
					}
					rolloutsEnabled = true
				} else {
					log.Warn(
						"Argo Rollouts integration was enabled, but no Argo Rollouts " +
							"CRDs were found. Proceeding without Argo Rollouts integration.",
					)
				}
			} else {
				log.Info("Argo Rollouts integration is disabled")
			}
			if err = kargoapi.AddToScheme(scheme); err != nil {
				return fmt.Errorf("add kargo api to scheme: %w", err)
			}

			internalClient, err := newClientForAPI(ctx, restCfg, scheme)
			if err != nil {
				return fmt.Errorf("create internal Kubernetes client: %w", err)
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
				return fmt.Errorf("create Kubernetes client: %w", err)
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

			srv := api.NewServer(cfg, kubeClient, internalClient, rolloutsEnabled)
			l, err := net.Listen(
				"tcp",
				fmt.Sprintf(
					"%s:%s",
					os.GetEnv("HOST", "0.0.0.0"),
					os.GetEnv("PORT", "8080"),
				),
			)
			if err != nil {
				return fmt.Errorf("error creating listener: %w", err)
			}
			defer l.Close()

			if err = srv.Serve(ctx, l); err != nil {
				return fmt.Errorf("serve: %w", err)
			}
			return nil
		},
	}
}

func newClientForAPI(ctx context.Context, r *rest.Config, scheme *runtime.Scheme) (client.Client, error) {
	mgr, err := ctrl.NewManager(r, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("new manager: %w", err)
	}

	// Index Promotions by Stage
	if err := kubeclient.IndexPromotionsByStage(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index Promotions by Stage: %w", err)
	}

	// Index Freight by Warehouse
	if err := kubeclient.IndexFreightByWarehouse(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index Freight by Warehouse: %w", err)
	}

	// Index Freight by Stages in which it has been verified
	if err := kubeclient.IndexFreightByVerifiedStages(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index Freight by Stages in which it has been verified: %w", err)
	}

	// Index Freight by Stages for which it is approved
	if err :=
		kubeclient.IndexFreightByApprovedStages(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index Freight by Stages for which it has been approved: %w", err)
	}

	// Index ServiceAccounts by ODIC email
	if err := kubeclient.IndexServiceAccountsByOIDCEmail(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index ServiceAccounts by OIDC email: %w", err)
	}
	// Index ServiceAccounts by OIDC groups
	if err := kubeclient.IndexServiceAccountsByOIDCGroups(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index ServiceAccounts by OIDC groups: %w", err)
	}
	// Index ServiceAccounts by OIDC subjects
	if err := kubeclient.IndexServiceAccountsByOIDCSubjects(ctx, mgr); err != nil {
		return nil, fmt.Errorf("index ServiceAccounts by OIDC subjects: %w", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(fmt.Errorf("start manager: %w", err))
		}
	}()

	return mgr.GetClient(), nil
}
