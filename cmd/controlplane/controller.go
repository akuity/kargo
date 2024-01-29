package main

import (
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
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
		RunE: func(cmd *cobra.Command, args []string) error {
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
					return errors.Wrap(
						err,
						"error loading REST config for Kargo controller manager",
					)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes core API to Kargo controller manager "+
							"scheme",
					)
				}
				if err = rollouts.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Argo Rollouts API to Kargo controller manager scheme",
					)
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kargo API to Kargo controller manager scheme",
					)
				}
				if kargoMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme: scheme,
						Metrics: server.Options{
							BindAddress: "0",
						},
					},
				); err != nil {
					return errors.Wrap(err, "error initializing Kargo controller manager")
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
					return errors.Wrap(
						err,
						"error loading REST config for Argo CD controller manager",
					)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				argocdNamespace := os.GetEnv("ARGOCD_NAMESPACE", "argocd")

				// There's a chance there is only permission to interact with Argo CD
				// Application resources in a single namespace, so we will use that
				// namespace when attempting to determine if Argo CD CRDs are installed.
				if argoCDExists(ctx, restCfg, argocdNamespace) {
					log.Info("Argo CD integration is enabled")
					scheme := runtime.NewScheme()
					if err = corev1.AddToScheme(scheme); err != nil {
						return errors.Wrap(
							err,
							"error adding Kubernetes core API to Argo CD controller "+
								"manager scheme",
						)
					}
					if err = argocd.AddToScheme(scheme); err != nil {
						return errors.Wrap(
							err,
							"error adding Argo CD API to Argo CD controller manager scheme",
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
						return errors.Wrap(
							err,
							"error initializing Argo CD Application controller manager",
						)
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

			var rolloutsMgr manager.Manager
			if types.MustParseBool(os.GetEnv("ROLLOUTS_INTEGRATION_ENABLED", "true")) {
				// If the env var is undefined, this will resolve to kubeconfig for the
				// cluster the controller is running in.
				//
				// It is typically defined if this controller is running somewhere other
				// than a cluster suitable for executing Argo Rollouts AnalysesRuns and
				// user-defined workloads. An example of this would be a a topology
				// wherein the Kargo and Argo CD controllers both run in management
				// clusters, that are a not suitable for for this purpose.
				restCfg, err :=
					kubernetes.GetRestConfig(ctx, os.GetEnv("ROLLOUTS_KUBECONFIG", ""))
				if err != nil {
					return errors.Wrap(
						err,
						"error loading REST config for Argo Rollouts controller manager",
					)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				if argoRolloutsExists(ctx, restCfg) {
					log.Info("Argo Rollouts integration is enabled")
					scheme := runtime.NewScheme()
					if err = rollouts.AddToScheme(scheme); err != nil {
						return errors.Wrap(
							err,
							"error adding Argo Rollouts API to Argo Rollouts controller "+
								"manager scheme",
						)
					}
					cacheOpts := cache.Options{} // Watches all namespaces by default
					if shardName != "" {
						// TODO: When NOT sharded, Kargo can simply create AnalysisRun
						// resources in the project namespaces. When sharded, AnalysisRun
						// resources must be created IN the shard clusters (not the Kargo
						// control plane cluster) and project namespaces do not exist in the
						// shard clusters. We need a place to put them, so for now we allow
						// the user to specify a namespace that that exists on each shard for
						// this purpose. Note that the namespace does not need to be the same
						// on every shard. This may be one of the weaker points in our tenancy
						// model and can stand to be improved.
						watchNamespace := os.GetEnv(
							"ARGO_ROLLOUTS_ANALYSIS_RUNS_NAMESPACE",
							"kargo-analysis-runs",
						)
						cacheOpts.DefaultNamespaces = map[string]cache.Config{
							watchNamespace: {},
						}
					}
					if rolloutsMgr, err = ctrl.NewManager(
						restCfg,
						ctrl.Options{
							Scheme: scheme,
							Metrics: server.Options{
								BindAddress: "0",
							},
							Cache: cacheOpts,
						},
					); err != nil {
						return errors.Wrap(
							err,
							"error initializing Argo Rollouts AnalysisRun controller manager",
						)
					}
				} else {
					log.Warn(
						"Argo Rollouts integration was enabled, but no Argo Rollouts " +
							"CRDs were found. Proceeding without Argo Rollouts integration.",
					)
				}
			} else {
				log.Info("Argo Rollouts integration is disabled")
			}

			var argocdClientForCreds client.Client
			if types.MustParseBool(
				os.GetEnv("ARGOCD_ENABLE_CREDENTIAL_BORROWING", "false"),
			) && argocdMgr != nil {
				argocdClientForCreds = argocdMgr.GetClient()
			}
			credentialsDB := credentials.NewKubernetesDatabase(
				kargoMgr.GetClient(),
				argocdClientForCreds,
				credentials.KubernetesDatabaseConfigFromEnv(),
			)

			if err := promotions.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				argocdMgr,
				credentialsDB,
				shardName,
			); err != nil {
				return errors.Wrap(err, "error setting up Promotions reconciler")
			}

			if err := stages.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				argocdMgr,
				rolloutsMgr,
				shardName,
			); err != nil {
				return errors.Wrap(err, "error setting up Stages reconciler")
			}

			if err := warehouses.SetupReconcilerWithManager(
				kargoMgr,
				credentialsDB,
				shardName,
			); err != nil {
				return errors.Wrap(err, "error setting up Warehouses reconciler")
			}

			var errChan = make(chan error)

			wg := sync.WaitGroup{}

			if argocdMgr != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := argocdMgr.Start(ctx); err != nil {
						errChan <- errors.Wrap(err, "error starting argo cd manager")
					}
				}()
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := kargoMgr.Start(ctx); err != nil {
					errChan <- errors.Wrap(err, "error starting kargo manager")
				}
			}()

			if rolloutsMgr != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := rolloutsMgr.Start(ctx); err != nil {
						errChan <- errors.Wrap(err, "error starting rollouts manager")
					}
				}()
			}

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
