package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
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

	PprofBindAddress string

	Logger *logging.Logger
}

func newWebhooksServerCommand() *cobra.Command {
	cmdOpts := &webhooksServerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
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
	o.PprofBindAddress = os.GetEnv("PPROF_BIND_ADDRESS", "")
}

func (o *webhooksServerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()
	o.Logger.Info(
		"Starting Kargo Webhooks Server",
		"version", version.Version,
		"commit", version.GitCommit,
		"GOMAXPROCS", runtime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)

	webhookCfg := libWebhook.ConfigFromEnv()

	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return fmt.Errorf("error getting REST config: %w", err)
	}

	scheme := rtime.NewScheme()
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
			PprofBindAddress: o.PprofBindAddress,
		},
	)
	if err != nil {
		return fmt.Errorf("new manager: %w", err)
	}

	// Index Stages by Freight
	if err = mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		indexer.StagesByFreightField,
		indexer.StagesByFreight,
	); err != nil {
		return fmt.Errorf("index Stages by Freight: %w", err)
	}

	if err = freight.SetupWebhookWithManager(ctx, webhookCfg, mgr); err != nil {
		return fmt.Errorf("setup Freight webhook: %w", err)
	}
	if err = project.SetupWebhookWithManager(
		mgr,
		project.WebhookConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("setup Project webhook: %w", err)
	}
	if err = promotion.SetupWebhookWithManager(ctx, webhookCfg, mgr); err != nil {
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
