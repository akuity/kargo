package cmd

import (
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/controller"
	"github.com/akuityio/kargo/internal/kubernetes"
	versionpkg "github.com/akuityio/kargo/internal/version"
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

			config, err := kargoConfig()
			if err != nil {
				return errors.Wrap(err, "new kargo config")
			}

			mgrCfg, err := ctrl.GetConfig()
			if err != nil {
				return errors.Wrap(err, "get controller config")
			}

			scheme := runtime.NewScheme()
			if err = clientgoscheme.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add kubernetes api to scheme")
			}
			if err = argocd.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add argocd api to scheme")
			}
			if err = api.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add kargo api to scheme")
			}
			mgr, err := ctrl.NewManager(
				mgrCfg,
				ctrl.Options{
					Scheme: scheme,
					Port:   9443,
				},
			)
			if err != nil {
				return errors.Wrap(err, "create manager")
			}

			kubeClient, err := kubernetes.Client()
			if err != nil {
				return errors.Wrap(err, "new kubernetes client")
			}
			argoDB := db.NewDB(
				"",
				// TODO: Do not hard-code the namespace
				settings.NewSettingsManager(ctx, kubeClient, "argo-cd"),
				kubeClient,
			)

			if err := (&api.Environment{}).SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "create webhook")
			}

			if err := controller.SetupEnvironmentReconcilerWithManager(
				ctx,
				config,
				mgr,
				kubeClient,
				argoDB,
				bookkeeper.NewService(
					&bookkeeper.ServiceOptions{
						LogLevel: bookkeeper.LogLevel(config.LogLevel),
					},
				),
			); err != nil {
				return errors.Wrap(err, "setup environment reconciler")
			}
			if err := controller.SetupPromotionReconcilerWithManager(
				ctx,
				config,
				mgr,
			); err != nil {
				return errors.Wrap(err, "setup promotion reconciler")
			}

			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				return errors.Wrap(err, "start controller")
			}
			return nil
		},
	}
}
