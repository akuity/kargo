package main

import (
	"context"
	"fmt"
	stdruntime "runtime"

	"github.com/spf13/cobra"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/types"
	libWebhook "github.com/akuity/kargo/internal/webhook/kubernetes"
	"github.com/akuity/kargo/internal/webhook/kubernetes/freight"
	"github.com/akuity/kargo/internal/webhook/kubernetes/project"
	"github.com/akuity/kargo/internal/webhook/kubernetes/projectconfig"
	"github.com/akuity/kargo/internal/webhook/kubernetes/promotion"
	"github.com/akuity/kargo/internal/webhook/kubernetes/promotiontask"
	"github.com/akuity/kargo/internal/webhook/kubernetes/stage"
	"github.com/akuity/kargo/internal/webhook/kubernetes/warehouse"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type kubernetesWebhooksServerOptions struct {
	KubeConfig string
	QPS        float32
	Burst      int

	MetricsBindAddress string
	PprofBindAddress   string

	Logger *logging.Logger
}

func newKubernetesWebhooksServerCommand() *cobra.Command {
	cmdOpts := &kubernetesWebhooksServerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
	}

	cmd := &cobra.Command{
		Use:               "kubernetes-webhooks-server",
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

func (o *kubernetesWebhooksServerOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
	o.QPS = types.MustParseFloat32(os.GetEnv("KUBE_API_QPS", "50.0"))
	o.Burst = types.MustParseInt(os.GetEnv("KUBE_API_BURST", "300"))

	o.MetricsBindAddress = os.GetEnv("METRICS_BIND_ADDRESS", "0")
	o.PprofBindAddress = os.GetEnv("PPROF_BIND_ADDRESS", "")
}

func (o *kubernetesWebhooksServerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()
	o.Logger.Info(
		"Starting Kargo Kubernetes Webhooks Server",
		"version", version.Version,
		"commit", version.GitCommit,
		"GOMAXPROCS", stdruntime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)

	webhookCfg := libWebhook.ConfigFromEnv()

	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return fmt.Errorf("error getting REST config: %w", err)
	}
	kubernetes.ConfigureQPSBurst(ctx, restCfg, o.QPS, o.Burst)

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
				BindAddress: o.MetricsBindAddress,
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
	if err = projectconfig.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("setup ProjectConfig webhook: %w", err)
	}
	if err = promotion.SetupWebhookWithManager(ctx, webhookCfg, mgr); err != nil {
		return fmt.Errorf("setup Promotion webhook: %w", err)
	}
	if err = promotiontask.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("setup PromotionTask webhook: %w", err)
	}
	if err = stage.SetupWebhookWithManager(webhookCfg, mgr); err != nil {
		return fmt.Errorf("setup Stage webhook: %w", err)
	}
	if err = warehouse.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("setup Warehouse webhook: %w", err)
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start Kargo Kubernetes webhook manager: %w", err)
	}
	return nil
}
