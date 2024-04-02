package main

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/promotions"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/stages"
	"github.com/akuity/kargo/internal/controller/warehouses"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/types"
	versionpkg "github.com/akuity/kargo/internal/version"

	_ "github.com/akuity/kargo/internal/gitprovider/github"
)

func newControllerCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "controller",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			shardName := os.GetEnv("SHARD_NAME", "")

			startupLogEntry := log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			})
			if shardName != "" {
				startupLogEntry = startupLogEntry.WithField("shard", shardName)
			}
			startupLogEntry.Info("Starting Kargo Controller")

			stagesReconcilerCfg := stages.ReconcilerConfigFromEnv()
			var kargoMgr manager.Manager
			{
				// If the env var is undefined, this will resolve to kubeconfig for the
				// cluster the controller is running in.
				//
				// It is typically defined if this controller is running somewhere other
				// than where the Kargo resources live. One example of this would be a
				// sharded topology wherein Kargo controllers run on application
				// clusters, with Kargo resources hosted in a centralized management
				// cluster.
				restCfg, err :=
					kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
				if err != nil {
					return fmt.Errorf("error loading REST config for Kargo controller manager: %w", err)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return fmt.Errorf(
						"error adding Kubernetes core API to Kargo controller manager scheme: %w",
						err,
					)
				}
				if err = extv1.AddToScheme(scheme); err != nil {
					return fmt.Errorf(
						"error adding Kubernetes API extensions API to Kargo controller manager scheme: %w",
						err,
					)
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return fmt.Errorf(
						"error adding Kargo API to Kargo controller manager scheme: %w",
						err,
					)
				}
				if stagesReconcilerCfg.RolloutsIntegrationEnabled {
					if argoRolloutsExists(ctx, restCfg) {
						log.Info("Argo Rollouts integration is enabled")
						if err = rollouts.AddToScheme(scheme); err != nil {
							return fmt.Errorf(
								"error adding Argo Rollouts API to Kargo controller manager scheme: %w",
								err,
							)
						}
					} else {
						log.Warn(
							"Argo Rollouts integration was enabled, but no Argo Rollouts " +
								"CRDs were found. Proceeding without Argo Rollouts integration.",
						)
					}
				}

				secretReq, err := controller.GetCredentialsRequirement()
				if err != nil {
					return fmt.Errorf("error getting label requirement for credentials Secrets: %w", err)
				}

				cacheOpts := cache.Options{
					ByObject: map[client.Object]cache.ByObject{
						// Only watch Secrets matching the label requirements
						// for credentials.
						&corev1.Secret{}: {
							Label: labels.NewSelector().Add(*secretReq),
						},
					},
				}

				if kargoMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme: scheme,
						Metrics: server.Options{
							BindAddress: "0",
						},
						Cache: cacheOpts,
					},
				); err != nil {
					return fmt.Errorf("error initializing Kargo controller manager: %w", err)
				}
			}

			var argocdMgr manager.Manager
			if types.MustParseBool(os.GetEnv("ARGOCD_INTEGRATION_ENABLED", "true")) {
				// If the env var is undefined, this will resolve to kubeconfig for the
				// cluster the controller is running in.
				//
				// It is typically defined if this controller is running somewhere other
				// than where the Argo CD resources live. Two examples of this would
				// involve topologies wherein Kargo controllers run EITHER sharded
				// across application clusters OR in a centralized management cluster,
				// but with Argo CD deployed to a different management cluster.
				restCfg, err :=
					kubernetes.GetRestConfig(ctx, os.GetEnv("ARGOCD_KUBECONFIG", ""))
				if err != nil {
					return fmt.Errorf("error loading REST config for Argo CD controller manager: %w", err)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				argocdNamespace := libargocd.Namespace()

				// There's a chance there is only permission to interact with Argo CD
				// Application resources in a single namespace, so we will use that
				// namespace when attempting to determine if Argo CD CRDs are installed.
				if argoCDExists(ctx, restCfg, argocdNamespace) {
					log.Info("Argo CD integration is enabled")
					scheme := runtime.NewScheme()
					if err = corev1.AddToScheme(scheme); err != nil {
						return fmt.Errorf(
							"error adding Kubernetes core API to Argo CD controller manager scheme: %w",
							err,
						)
					}
					if err = argocd.AddToScheme(scheme); err != nil {
						return fmt.Errorf(
							"error adding Argo CD API to Argo CD controller manager scheme: %w",
							err,
						)
					}
					cacheOpts := cache.Options{} // Watches all namespaces by default
					if types.MustParseBool(
						os.GetEnv("ARGOCD_WATCH_ARGOCD_NAMESPACE_ONLY", "false"),
					) {
						cacheOpts.DefaultNamespaces = map[string]cache.Config{
							argocdNamespace: {},
						}
					}
					if argocdMgr, err = ctrl.NewManager(
						restCfg,
						ctrl.Options{
							Scheme: scheme,
							Metrics: server.Options{
								BindAddress: "0",
							},
							Cache: cacheOpts,
						},
					); err != nil {
						return fmt.Errorf("error initializing Argo CD Application controller manager: %w", err)
					}
				} else {
					log.Warn(
						"ARGO CD integration was enabled, but no Argo CD CRDs were " +
							"found. Proceeding without Argo CD integration.",
					)
				}
			} else {
				log.Info("Argo CD integration is disabled")
			}

			credentialsDB := credentials.NewKubernetesDatabase(
				kargoMgr.GetClient(),
				credentials.KubernetesDatabaseConfigFromEnv(),
			)

			if err := promotions.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				argocdMgr,
				credentialsDB,
				shardName,
			); err != nil {
				return fmt.Errorf("error setting up Promotions reconciler: %w", err)
			}

			if err := stages.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				argocdMgr,
				stagesReconcilerCfg,
			); err != nil {
				return fmt.Errorf("error setting up Stages reconciler: %w", err)
			}

			if err := warehouses.SetupReconcilerWithManager(
				kargoMgr,
				credentialsDB,
				shardName,
			); err != nil {
				return fmt.Errorf("error setting up Warehouses reconciler: %w", err)
			}

			var errChan = make(chan error)

			wg := sync.WaitGroup{}

			if argocdMgr != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := argocdMgr.Start(ctx); err != nil {
						errChan <- fmt.Errorf("error starting argo cd manager: %w", err)
					}
				}()
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := kargoMgr.Start(ctx); err != nil {
					errChan <- fmt.Errorf("error starting kargo manager: %w", err)
				}
			}()

			// Adapt wg to a channel that can be used in a select
			doneCh := make(chan struct{})
			go func() {
				wg.Wait()
				close(doneCh)
			}()

			select {
			case err := <-errChan:
				return err
			case <-doneCh:
				return nil
			}
		},
	}
}
