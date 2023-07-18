package main

import (
	"sync"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/applications"
	"github.com/akuity/kargo/internal/controller/promotions"
	"github.com/akuity/kargo/internal/controller/stages"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/types"
	versionpkg "github.com/akuity/kargo/internal/version"
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
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo Controller")

			var kargoMgr manager.Manager
			{
				restCfg, err := getRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
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
				if err = api.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kargo API to Kargo controller manager scheme",
					)
				}
				if kargoMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme:             scheme,
						MetricsBindAddress: "0",
					},
				); err != nil {
					return errors.Wrap(err, "error initializing Kargo controller manager")
				}
			}

			var appMgr manager.Manager
			{
				restCfg, err := getRestConfig(ctx, os.GetEnv("ARGOCD_KUBECONFIG", ""))
				if err != nil {
					return errors.Wrap(
						err,
						"error loading REST config for Argo CD Application controller "+
							"manager",
					)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes core API to Argo CD Application "+
							"controller manager scheme",
					)
				}
				if err = argocd.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kargo API to Argo CD Application controller manager "+
							"scheme",
					)
				}

				var watchNamespace string // Empty string means all namespaces
				if types.MustParseBool(
					os.GetEnv("ARGOCD_WATCH_ARGOCD_NAMESPACE_ONLY", "false"),
				) {
					watchNamespace = os.GetEnv("ARGOCD_NAMESPACE", "argocd")
				}
				if appMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme:             scheme,
						MetricsBindAddress: "0",
						Namespace:          watchNamespace,
					},
				); err != nil {
					return errors.Wrap(
						err,
						"error initializing Argo CD Application controller manager",
					)
				}
			}

			var argoClientForCreds client.Client
			if types.MustParseBool(
				os.GetEnv("ARGOCD_ENABLE_CREDENTIAL_BORROWING", "false"),
			) {
				argoClientForCreds = appMgr.GetClient()
			}
			credentialsDB := credentials.NewKubernetesDatabase(
				os.GetEnv("ARGOCD_NAMESPACE", "argocd"),
				kargoMgr.GetClient(),
				argoClientForCreds,
			)

			if err := stages.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				appMgr,
				credentialsDB,
			); err != nil {
				return errors.Wrap(err, "error setting up Stages reconciler")
			}

			if err := promotions.SetupReconcilerWithManager(
				kargoMgr,
				appMgr,
				credentialsDB,
				bookkeeper.NewService(
					&bookkeeper.ServiceOptions{
						LogLevel: bookkeeper.LogLevel(logging.LoggerFromContext(ctx).Level),
					},
				),
			); err != nil {
				return errors.Wrap(err, "error setting up Promotions reconciler")
			}

			if err :=
				applications.SetupReconcilerWithManager(kargoMgr, appMgr); err != nil {
				return errors.Wrap(err, "error setting up Applications reconciler")
			}

			var errChan = make(chan error)

			wg := sync.WaitGroup{}

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := kargoMgr.Start(ctx); err != nil {
					errChan <- errors.Wrap(err, "error starting kargo manager")
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := appMgr.Start(ctx); err != nil {
					errChan <- errors.Wrap(err, "error starting argo manager")
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
