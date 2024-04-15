package main

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
	libWebhook "github.com/akuity/kargo/internal/webhook"
	"github.com/akuity/kargo/internal/webhook/freight"
	"github.com/akuity/kargo/internal/webhook/project"
	"github.com/akuity/kargo/internal/webhook/promotion"
	"github.com/akuity/kargo/internal/webhook/stage"
	"github.com/akuity/kargo/internal/webhook/warehouse"
)

type webhooksServerOptions struct {
	KubeConfig string

	Logger *log.Logger
}

func newWebhooksServerCommand() *cobra.Command {
	cmdOpts := &webhooksServerOptions{
		Logger: log.StandardLogger(),
	}

	cmd := &cobra.Command{
		Use:               "webhooks-server",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdOpts.complete()

			return cmdOpts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *webhooksServerOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
}

func (o *webhooksServerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()
	o.Logger.WithFields(log.Fields{
		"version": version.Version,
		"commit":  version.GitCommit,
	}).Info("Starting Kargo Webhooks Server")

	webhookCfg := libWebhook.ConfigFromEnv()

	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return fmt.Errorf("error getting REST config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add corev1 to scheme: %w", err)
	}
	if err = rbacv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add rbacv1 to scheme: %w", err)
	}
	if err = authzv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add authzv1 to scheme: %w", err)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add kargo api to scheme: %w", err)
	}

	mgr, err := ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			WebhookServer: webhook.NewServer(
				webhook.Options{
					Port: 9443,
				},
			),
			Metrics: server.Options{
				BindAddress: "0",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("new manager: %w", err)
	}

	// Index Stages by Freight
	if err = kubeclient.IndexStagesByFreight(ctx, mgr); err != nil {
		return fmt.Errorf("index Stages by Freight: %w", err)
	}

	if err = freight.SetupWebhookWithManager(webhookCfg, mgr); err != nil {
		return fmt.Errorf("setup Freight webhook: %w", err)
	}
	if err = project.SetupWebhookWithManager(
		mgr,
		project.WebhookConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("setup Project webhook: %w", err)
	}
	if err = promotion.SetupWebhookWithManager(webhookCfg, mgr); err != nil {
		return fmt.Errorf("setup Promotion webhook: %w", err)
	}
	if err = stage.SetupWebhookWithManager(webhookCfg, mgr); err != nil {
		return fmt.Errorf("setup Stage webhook: %w", err)
	}
	if err = warehouse.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("setup Warehouse webhook: %w", err)
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start Kargo webhook manager: %w", err)
	}
	return nil
}
