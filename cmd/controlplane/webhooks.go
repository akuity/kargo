package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
	"github.com/akuity/kargo/internal/webhook/freight"
	"github.com/akuity/kargo/internal/webhook/promotion"
	"github.com/akuity/kargo/internal/webhook/promotionpolicy"
	"github.com/akuity/kargo/internal/webhook/stage"
	"github.com/akuity/kargo/internal/webhook/warehouse"
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

			scheme := runtime.NewScheme()
			if err = corev1.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add corev1 to scheme")
			}
			if err = authzv1.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add authzv1 to scheme")
			}
			if err = kargoapi.AddToScheme(scheme); err != nil {
				return errors.Wrap(err, "add kargo api to scheme")
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
			if err = freight.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "setup Freight webhook")
			}
			if err = warehouse.SetupWebhookWithManager(mgr); err != nil {
				return errors.Wrap(err, "setup Warehouse webhook")
			}

			return errors.Wrap(
				mgr.Start(ctx),
				"start Kargo webhook manager",
			)
		},
	}
}
