package main

import (
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/controller/environments"
	"github.com/akuity/kargo/internal/controller/promotions"
	"github.com/akuity/kargo/internal/credentials"
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

			config := config.NewControllerConfig()
			mgrCfg, err := ctrl.GetConfig()
			if err != nil {
				return errors.Wrap(err, "get controller config")
			}

			scheme := runtime.NewScheme()
			if err = corev1.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add kubernetes core api to scheme")
			}
			if err = rbacv1.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add kubernetes rbac api to scheme")
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

			if err = environments.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "error initializing Environment webhooks")
			}
			if err =
				promotions.SetupWebhookWithManager(ctx, mgr, config); err != nil {
				return errors.Wrap(err, "error initializing Environment webhooks")
			}

			credentialsDB, err :=
				credentials.NewKubernetesDatabase(ctx, config.ArgoCDNamespace, mgr)
			if err != nil {
				return errors.Wrap(err, "error initializing credentials DB")
			}

			if err := environments.SetupReconcilerWithManager(
				ctx,
				mgr,
				credentialsDB,
			); err != nil {
				return errors.Wrap(err, "setup environment reconciler")
			}
			if err := promotions.SetupReconcilerWithManager(
				mgr,
				credentialsDB,
				bookkeeper.NewService(
					&bookkeeper.ServiceOptions{
						LogLevel: bookkeeper.LogLevel(config.LogLevel),
					},
				),
			); err != nil {
				return errors.Wrap(err, "setup promotion reconciler")
			}

			if err := mgr.Start(ctx); err != nil {
				return errors.Wrap(err, "start controller")
			}
			return nil
		},
	}
}
