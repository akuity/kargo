package main

import (
	"context"
	"fmt"
	stdruntime "runtime"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/management/clusterconfigs"
	"github.com/akuity/kargo/pkg/controller/management/namespaces"
	"github.com/akuity/kargo/pkg/controller/management/projectconfigs"
	"github.com/akuity/kargo/pkg/controller/management/projects"
	"github.com/akuity/kargo/pkg/controller/management/secrets"
	"github.com/akuity/kargo/pkg/controller/management/serviceaccounts"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/types"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type managementControllerOptions struct {
	KubeConfig string
	QPS        float32
	Burst      int

	KargoNamespace               string
	ManageControllerRoleBindings bool

	MetricsBindAddress string
	PprofBindAddress   string

	Logger *logging.Logger
}

func newManagementControllerCommand() *cobra.Command {
	cmdOpts := &managementControllerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLoggerOrDie(logging.InfoLevel, logging.DefaultFormat),
	}

	cmd := &cobra.Command{
		Use:               "management-controller",
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

func (o *managementControllerOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
	o.QPS = types.MustParseFloat32(os.GetEnv("KUBE_API_QPS", "50.0"))
	o.Burst = types.MustParseInt(os.GetEnv("KUBE_API_BURST", "300"))

	o.KargoNamespace = os.GetEnv("KARGO_NAMESPACE", "kargo")
	o.ManageControllerRoleBindings = types.MustParseBool(os.GetEnv("MANAGE_CONTROLLER_ROLE_BINDINGS", "true"))

	o.MetricsBindAddress = os.GetEnv("METRICS_BIND_ADDRESS", "0")
	o.PprofBindAddress = os.GetEnv("PPROF_BIND_ADDRESS", "")
}

func (o *managementControllerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()

	o.Logger.Info(
		"Starting Kargo Management Controller",
		"version", version.Version,
		"commit", version.GitCommit,
		"GOMAXPROCS", stdruntime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)

	systemResourcesCfg := secrets.ReconcilerConfig{
		ControllerName:       "system-resources-migration-controller",
		SourceNamespace:      os.GetEnv("CLUSTER_SECRETS_NAMESPACE", "kargo-cluster-secrets"),
		DestinationNamespace: os.GetEnv("SYSTEM_RESOURCES_NAMESPACE", "kargo-system-resources"),
	}

	sharedResourcesCfg := secrets.ReconcilerConfig{
		ControllerName:       "shared-resources-migration-controller",
		SourceNamespace:      os.GetEnv("GLOBAL_CREDENTIALS_NAMESPACE", ""),
		DestinationNamespace: os.GetEnv("SHARED_RESOURCES_NAMESPACE", "kargo-shared-resources"),
	}

	kargoMgr, err := o.setupManager(ctx, systemResourcesCfg, sharedResourcesCfg)
	if err != nil {
		return fmt.Errorf("error initializing Kargo controller manager: %w", err)
	}

	if err := clusterconfigs.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		clusterconfigs.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up ClusterConfigs reconciler: %w", err)
	}

	if err := namespaces.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		namespaces.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up Namespaces reconciler: %w", err)
	}

	if err := projects.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		projects.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up Projects reconciler: %w", err)
	}

	if err := projectconfigs.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		projectconfigs.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up ProjectConfigs reconciler: %w", err)
	}

	if o.ManageControllerRoleBindings {
		if err := serviceaccounts.SetupReconcilerWithManager(
			ctx,
			kargoMgr,
			serviceaccounts.ReconcilerConfigFromEnv(),
		); err != nil {
			return fmt.Errorf("error setting up ServiceAccount reconciler: %w", err)
		}
	}

	if systemResourcesCfg.SourceNamespace != "" &&
		systemResourcesCfg.SourceNamespace != systemResourcesCfg.DestinationNamespace {
		if err := secrets.SetupReconcilerWithManager(
			ctx,
			kargoMgr,
			systemResourcesCfg,
		); err != nil {
			return fmt.Errorf("error setting up Secrets reconciler for system resources namespace: %w", err)
		}
	}

	if sharedResourcesCfg.SourceNamespace != "" &&
		sharedResourcesCfg.SourceNamespace != sharedResourcesCfg.DestinationNamespace {
		if err := secrets.SetupReconcilerWithManager(
			ctx,
			kargoMgr,
			sharedResourcesCfg,
		); err != nil {
			return fmt.Errorf("error setting up Secrets reconciler for shared resources namespace: %w", err)
		}
	}

	if err := kargoMgr.Start(ctx); err != nil {
		return fmt.Errorf("error starting kargo manager: %w", err)
	}
	return nil
}

func (o *managementControllerOptions) setupManager(
	ctx context.Context,
	systemResourcesCfg secrets.ReconcilerConfig,
	sharedResourcesCfg secrets.ReconcilerConfig,
) (manager.Manager, error) {
	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading REST config for Kargo controller manager: %w", err)
	}
	kubernetes.ConfigureQPSBurst(ctx, restCfg, o.QPS, o.Burst)
	restCfg.ContentType = runtime.ContentTypeJSON

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kubernetes core API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = rbacv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kubernetes RBAC API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kargo API to Kargo controller manager scheme: %w",
			err,
		)
	}
	namespaceCacheConfigs := make(map[string]cache.Config)
	if systemResourcesCfg.SourceNamespace != "" {
		namespaceCacheConfigs[systemResourcesCfg.SourceNamespace] = cache.Config{}
		namespaceCacheConfigs[systemResourcesCfg.DestinationNamespace] = cache.Config{}
	}
	if sharedResourcesCfg.SourceNamespace != "" {
		namespaceCacheConfigs[sharedResourcesCfg.SourceNamespace] = cache.Config{}
		namespaceCacheConfigs[sharedResourcesCfg.DestinationNamespace] = cache.Config{}
	}
	return ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: o.MetricsBindAddress,
			},
			PprofBindAddress: o.PprofBindAddress,
			Cache: cache.Options{
				ByObject: map[client.Object]cache.ByObject{
					&corev1.ServiceAccount{}: {
						Namespaces: map[string]cache.Config{
							o.KargoNamespace: {},
						},
					},
					&corev1.Secret{}: {
						Namespaces: namespaceCacheConfigs,
					},
				},
			},
		},
	)
}
