package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
	"github.com/akuity/kargo/internal/webhook/promotion"
	"github.com/akuity/kargo/internal/webhook/promotionpolicy"
	"github.com/akuity/kargo/internal/webhook/stage"
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

			restCfg, err := kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
			if err != nil {
				return errors.Wrap(err, "error getting REST config")
			}

			scheme, err := kubeclient.NewWebhooksServerScheme()
			if err != nil {
				return errors.Wrap(err, "new webhooks server scheme")
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
				return errors.Wrap(err, "new manager")
			}

			// Index PromotionPolicies by Stage
			if err = kubeclient.IndexPromotionPoliciesByStage(ctx, mgr); err != nil {
				return errors.Wrap(err, "index PromotionPolicies by Stage")
			}

			if err = stage.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "setup Stage webhook")
			}
			if err = promotion.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "setup Promotion webhook")
			}
			if err = promotionpolicy.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "setup PromotionPolicy webhook")
			}

			return errors.Wrap(
				mgr.Start(ctx),
				"start Kargo webhook manager",
			)
		},
	}
}
