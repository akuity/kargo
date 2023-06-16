package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/akuity/kargo/api/v1alpha1"
	libConfig "github.com/akuity/kargo/internal/config"
	versionpkg "github.com/akuity/kargo/internal/version"
	"github.com/akuity/kargo/internal/webhooks/environments"
	"github.com/akuity/kargo/internal/webhooks/promotions"
)

func newWebhooksServerCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "webhooks-server",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo Webhooks Server")

			webhooksCfg := libConfig.NewWebhooksConfig()

			restCfg, _, err := getMgrConfigs()
			if err != nil {
				return errors.Wrap(err, "error getting REST config")
			}

			scheme := runtime.NewScheme()
			if err = rbacv1.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "error adding rbacv1 to scheme")
			}
			if err = api.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "error adding Kargo api to scheme")
			}

			mgr, err := ctrl.NewManager(
				restCfg,
				ctrl.Options{
					Scheme:             scheme,
					MetricsBindAddress: "0",
					Port:               9443,
				},
			)
			if err != nil {
				return errors.Wrap(err, "error creating webhooks manager")
			}

			if err = environments.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "error initializing Environment webhooks")
			}
			if err =
				promotions.SetupWebhookWithManager(ctx, mgr, webhooksCfg); err != nil {
				return errors.Wrap(err, "error initializing Promotion webhooks")
			}

			return errors.Wrap(
				mgr.Start(ctx),
				"error starting Kargo webhooks manager",
			)
		},
	}
}
