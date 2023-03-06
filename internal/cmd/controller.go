package cmd

import (
	"fmt"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/argoproj/argo-cd/v2/util/settings"
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
				return fmt.Errorf("new kargo config: %w", err)
			}

			mgrCfg, err := ctrl.GetConfig()
			if err != nil {
				return fmt.Errorf("get controller config: %w", err)
			}

			scheme := runtime.NewScheme()
			if err = clientgoscheme.AddToScheme(scheme); err != nil {
				return fmt.Errorf("add kubernetes api to scheme: %w", err)
			}
			if err = argocd.AddToScheme(scheme); err != nil {
				return fmt.Errorf("add argocd api to scheme: %w", err)
			}
			if err = api.AddToScheme(scheme); err != nil {
				return fmt.Errorf("add kargo api to scheme: %w", err)
			}
			mgr, err := ctrl.NewManager(
				mgrCfg,
				ctrl.Options{
					Scheme: scheme,
					Port:   9443,
				},
			)
			if err != nil {
				return fmt.Errorf("create manager: %w", err)
			}

			kubeClient, err := kubernetes.Client()
			if err != nil {
				return fmt.Errorf("new kubernetes client: %w", err)
			}
			argoDB := db.NewDB(
				"",
				// TODO: Do not hard-code the namespace
				settings.NewSettingsManager(ctx, kubeClient, "argo-cd"),
				kubeClient,
			)

			if err := (&api.Environment{}).SetupWebhookWithManager(mgr); err != nil {
				return fmt.Errorf("create webhook: %w", err)
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
				return fmt.Errorf("setup environment reconciler: %w", err)
			}
			if err := controller.SetupPromotionReconcilerWithManager(
				ctx,
				config,
				mgr,
			); err != nil {
				return fmt.Errorf("setup promotion reconciler: %w", err)
			}

			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				return fmt.Errorf("start controller: %w", err)
			}
			return nil
		},
	}
}
