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
	"github.com/akuity/kargo/pkg/controller/management/dbsync"
	"github.com/akuity/kargo/pkg/controller/management/legacysecrets"
	"github.com/akuity/kargo/pkg/controller/management/namespaces"
	"github.com/akuity/kargo/pkg/controller/management/projectconfigs"
	"github.com/akuity/kargo/pkg/controller/management/projects"
	"github.com/akuity/kargo/pkg/controller/management/replication"
	"github.com/akuity/kargo/pkg/controller/management/serviceaccounts"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/types"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type managementControllerOptions struct {
	DatabaseURL string
	KubeConfig  string
	QPS         float32
	Burst       int

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
	o.DatabaseURL = os.GetEnv("DATABASE_URL", "")
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
	o.QPS = types.MustParseFloat32(os.GetEnv("KUBE_API_QPS", "50.0"))
	o.Burst = types.MustParseInt(os.GetEnv("KUBE_API_BURST", "300"))

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

	natsConn, err := connectNATS(ctx, o.Logger)
	if err != nil {
		return err
	}
	defer natsConn.Close()

	kargoMgr, err := o.setupManager(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Kargo controller manager: %w", err)
	}

	// One-time cleanup of a hazard left behind by the removal of the
	// automatic Secret migration reconciler. See the package doc comment
	// on legacysecrets for details.
	if err := legacysecrets.RemoveOrphanedFinalizers(
		ctx,
		kargoMgr.GetAPIReader(),
		kargoMgr.GetClient(),
		os.GetEnv("SYSTEM_RESOURCES_NAMESPACE", "kargo-system-resources"),
		os.GetEnv("SHARED_RESOURCES_NAMESPACE", "kargo-shared-resources"),
	); err != nil {
		return fmt.Errorf("error cleaning up orphaned Secret finalizers: %w", err)
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

	replicationCfg := replication.ReconcilerConfig{
		SharedResourcesNamespace: os.GetEnv("SHARED_RESOURCES_NAMESPACE", "kargo-shared-resources"),
		MaxConcurrentReconciles:  4,
	}
	if err := replication.SetupSecretReconcilerWithManager(ctx, kargoMgr, replicationCfg); err != nil {
		return fmt.Errorf("error setting up shared Secret replication reconciler: %w", err)
	}
	if err := replication.SetupConfigMapReconcilerWithManager(ctx, kargoMgr, replicationCfg); err != nil {
		return fmt.Errorf("error setting up shared ConfigMap replication reconciler: %w", err)
	}

	if o.DatabaseURL != "" {
		pool, poolErr := database.NewPool(ctx, o.DatabaseURL)
		if poolErr != nil {
			return fmt.Errorf("error configuring database synchronization: %w", poolErr)
		}
		defer pool.Close()
		if err := dbsync.SetupWithManager(ctx, kargoMgr, database.NewStore(pool)); err != nil {
			return err
		}
	}

	if err := kargoMgr.Start(ctx); err != nil {
		return fmt.Errorf("error starting kargo manager: %w", err)
	}
	return nil
}

func (o *managementControllerOptions) setupManager(
	ctx context.Context,
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
	namespaceCacheConfigs := map[string]cache.Config{
		os.GetEnv("SYSTEM_RESOURCES_NAMESPACE", "kargo-system-resources"): {},
		os.GetEnv("SHARED_RESOURCES_NAMESPACE", "kargo-shared-resources"): {},
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
					&corev1.ServiceAccount{}: {},
					&corev1.Secret{}: {
						Namespaces: namespaceCacheConfigs,
					},
					&corev1.ConfigMap{}: {
						Namespaces: namespaceCacheConfigs,
					},
				},
			},
		},
	)
}
