package main

import (
	"context"
	"fmt"
	stdruntime "runtime"
	"sync"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	libCluster "sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
	"github.com/akuity/kargo/pkg/controller"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/promotions"
	"github.com/akuity/kargo/pkg/controller/stages"
	"github.com/akuity/kargo/pkg/controller/warehouses"
	"github.com/akuity/kargo/pkg/credentials"
	credsdb "github.com/akuity/kargo/pkg/credentials/kubernetes"
	"github.com/akuity/kargo/pkg/health"
	healthCheckers "github.com/akuity/kargo/pkg/health/checker/builtin"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/subscription"
	"github.com/akuity/kargo/pkg/types"
	versionpkg "github.com/akuity/kargo/pkg/x/version"

	_ "github.com/akuity/kargo/pkg/credentials/acr"
	_ "github.com/akuity/kargo/pkg/credentials/basic"
	_ "github.com/akuity/kargo/pkg/credentials/ecr"
	_ "github.com/akuity/kargo/pkg/credentials/gar"
	_ "github.com/akuity/kargo/pkg/credentials/github"
	_ "github.com/akuity/kargo/pkg/credentials/ssh"
	_ "github.com/akuity/kargo/pkg/promotion/runner/builtin"
)

type controllerOptions struct {
	IsDefaultController bool
	ShardName           string

	ControlPlaneKubeConfig string
	QPS                    float32
	Burst                  int

	ArgoCDEnabled       bool
	ArgoCDKubeConfig    string
	ArgoCDNamespaceOnly bool

	MetricsBindAddress string
	PprofBindAddress   string

	Logger *logging.Logger
}

func newControllerCommand() *cobra.Command {
	_, format := getLogVars()
	cmdOpts := &controllerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLoggerOrDie(logging.InfoLevel, format),
	}

	cmd := &cobra.Command{
		Use:               "controller",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			version := versionpkg.GetVersion()
			startupLogger := cmdOpts.Logger.WithValues(
				"version", version.Version,
				"commit", version.GitCommit,
				"GOMAXPROCS", stdruntime.GOMAXPROCS(0),
				"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
				"defaultController", cmdOpts.IsDefaultController,
			)
			if cmdOpts.ShardName != "" {
				startupLogger = startupLogger.WithValues("shard", cmdOpts.ShardName)
			}
			startupLogger.Info("Starting Kargo Controller")
			cmdOpts.complete()

			return cmdOpts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *controllerOptions) complete() {
	o.IsDefaultController = types.MustParseBool(os.GetEnv("IS_DEFAULT_CONTROLLER", "false"))
	o.ShardName = os.GetEnv("SHARD_NAME", "")

	o.ControlPlaneKubeConfig = os.GetEnv("KUBECONFIG", "")
	o.QPS = types.MustParseFloat32(os.GetEnv("KUBE_API_QPS", "50.0"))
	o.Burst = types.MustParseInt(os.GetEnv("KUBE_API_BURST", "300"))

	o.ArgoCDEnabled = types.MustParseBool(os.GetEnv("ARGOCD_INTEGRATION_ENABLED", "true"))
	o.ArgoCDKubeConfig = os.GetEnv("ARGOCD_KUBECONFIG", "")
	o.ArgoCDNamespaceOnly = types.MustParseBool(os.GetEnv("ARGOCD_WATCH_ARGOCD_NAMESPACE_ONLY", "false"))

	o.MetricsBindAddress = os.GetEnv("METRICS_BIND_ADDRESS", "0")
	o.PprofBindAddress = os.GetEnv("PPROF_BIND_ADDRESS", "")

	logLevel, logFormat := getLogVars()

	o.Logger = logging.NewLoggerOrDie(logLevel, logFormat)
}

func (o *controllerOptions) run(ctx context.Context) error {
	kargoMgr, localClusterClient, stagesReconcilerCfg, err := o.setupKargoManager(
		ctx,
		stages.ReconcilerConfigFromEnv(),
	)
	if err != nil {
		return fmt.Errorf("error initializing Kargo controller manager: %w", err)
	}

	argocdMgr, err := o.setupArgoCDManager(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Argo CD Application controller manager: %w", err)
	}

	credentialsDB := credsdb.NewDatabase(
		kargoMgr.GetClient(),
		localClusterClient,
		credentials.DefaultProviderRegistry,
		credsdb.DatabaseConfigFromEnv(),
	)

	if err := o.setupReconcilers(
		ctx,
		kargoMgr,
		argocdMgr,
		credentialsDB,
		stagesReconcilerCfg,
	); err != nil {
		return fmt.Errorf("error setting up reconcilers: %w", err)
	}

	return o.startManagers(ctx, kargoMgr, argocdMgr)
}

func (o *controllerOptions) setupKargoManager(
	ctx context.Context,
	stagesReconcilerCfg stages.ReconcilerConfig,
) (manager.Manager, client.Client, stages.ReconcilerConfig, error) {
	// If o.ControlPlaneKubeConfig is empty, this will resolve to kubeconfig for
	// the cluster the controller is running in.
	//
	// It is typically non-empty only when this controller is running somewhere
	// other than the Kargo control plane's cluster. i.e. Running in an
	// application cluster, as part of a sharded topology.
	restCfg, err := kubernetes.GetRestConfig(ctx, o.ControlPlaneKubeConfig)
	if err != nil {
		return nil, nil, stagesReconcilerCfg,
			fmt.Errorf("error loading REST config for Kargo controller manager: %w", err)
	}
	kubernetes.ConfigureQPSBurst(ctx, restCfg, o.QPS, o.Burst)
	restCfg.ContentType = runtime.ContentTypeJSON

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, nil, stagesReconcilerCfg, fmt.Errorf(
			"error adding Kubernetes core API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, nil, stagesReconcilerCfg, fmt.Errorf(
			"error adding Kargo API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if stagesReconcilerCfg.RolloutsIntegrationEnabled {
		var exists bool
		if exists, err = argoRolloutsExists(ctx, restCfg); exists {
			o.Logger.Info("Argo Rollouts integration is enabled")
			if err = rollouts.AddToScheme(scheme); err != nil {
				return nil, nil, stagesReconcilerCfg, fmt.Errorf(
					"error adding Argo Rollouts API to Kargo controller manager scheme: %w",
					err,
				)
			}
		} else {
			// If we are unable to determine if Argo Rollouts is installed, we
			// will return an error and fail to start the controller. Note this
			// will only happen if we get an inconclusive response from the API
			// server (e.g. due to network issues), and not if Argo Rollouts is
			// not installed.
			if err != nil {
				return nil, nil, stagesReconcilerCfg, fmt.Errorf(
					"unable to determine if Argo Rollouts is installed: %w",
					err,
				)
			}

			// Disable Argo Rollouts integration if the CRDs are not found.
			stagesReconcilerCfg.RolloutsIntegrationEnabled = false
			o.Logger.Info(
				"Argo Rollouts integration was enabled, but no Argo Rollouts " +
					"CRDs were found. Proceeding without Argo Rollouts integration.",
			)
		}
	}

	// We may or may not be able to distill stagesReconcilerCfg.ShardName and
	// stagesReconcilerCfg.IsDefaultController down to a labels.Requirement. If
	// we're able to do so, we'll build a labels.Selector from that requirement
	// and use it to narrow the set of Stages and Promotions our client's internal
	// cache needs to be concerned with watching. If we're unable to distill
	// stagesReconcilerCfg.ShardName and stagesReconcilerCfg.IsDefaultController
	// down to a labels.Requirement, then we have no choice but to let our
	// client's internal cache watch all Stages and Promotions and the respective
	// reconcilers for those types will, instead, need to apply a predicate to
	// filter out resources for which they are not responsible.
	cacheOpts := cache.Options{}
	shardReq, err := controller.GetShardRequirement(
		stagesReconcilerCfg.ShardName,
		stagesReconcilerCfg.IsDefaultController,
	)
	if err != nil {
		return nil, nil, stagesReconcilerCfg,
			fmt.Errorf("error getting shard requirement: %w", err)
	}
	if shardReq != nil {
		shardSelector := labels.NewSelector().Add(*shardReq)
		cacheOpts.ByObject = map[client.Object]cache.ByObject{
			&kargoapi.Stage{}:     {Label: shardSelector},
			&kargoapi.Promotion{}: {Label: shardSelector},
		}
	}

	mgr, err := ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: o.MetricsBindAddress,
			},
			PprofBindAddress: o.PprofBindAddress,
			Client: client.Options{
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						// The controller does not have cluster-wide permissions, to
						// get/list/watch Secrets. Its access to Secrets grows and shrinks
						// dynamically as Projects are created and deleted. We disable
						// caching here since the underlying informer will not be able to
						// watch Secrets in all namespaces.
						&corev1.Secret{},
						// The controller has cluster-wide permissions to get/list/watch
						// ConfigMaps, but ConfigMaps have the potential to be quite large,
						// so we prefer to not cache them.
						&corev1.ConfigMap{},
					},
				},
			},
			Cache: cacheOpts,
		},
	)

	// If the mgr happens to be for the local cluster, or falling back to the
	// local cluster when searching for credentials is not explicitly enabled,
	// we're all done.
	if o.ControlPlaneKubeConfig == "" ||
		os.GetEnv("LOCAL_CLUSTER_CREDS_FALLBACK", "false") != "true" {
		return mgr, nil, stagesReconcilerCfg, err
	}

	// Build a separate client for the local cluster...
	if restCfg, err = kubernetes.GetRestConfig(ctx, ""); err != nil {
		return nil, nil, stagesReconcilerCfg,
			fmt.Errorf("error loading REST config for local cluster client: %w", err)
	}

	scheme = runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, nil, stagesReconcilerCfg, fmt.Errorf(
			"error adding Kubernetes core API to local client scheme: %w",
			err,
		)
	}
	localCluster, err := libCluster.New(
		restCfg,
		func(clusterOptions *libCluster.Options) {
			clusterOptions.Scheme = scheme
			clusterOptions.Client = client.Options{
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						// The controller does not have cluster-wide permissions, to
						// get/list/watch Secrets. Its access to Secrets grows and shrinks
						// dynamically as Projects are created and deleted. We disable
						// caching here since the underlying informer will not be able to
						// watch Secrets in all namespaces.
						&corev1.Secret{},
					},
				},
			}
		},
	)
	if err != nil {
		return nil, nil, stagesReconcilerCfg,
			fmt.Errorf("error creating Kubernetes client for local cluster: %w", err)
	}

	go func() {
		err = localCluster.Start(ctx)
	}()
	if !localCluster.GetCache().WaitForCacheSync(ctx) {
		return nil, nil, stagesReconcilerCfg,
			fmt.Errorf("error waiting for cache to sync: %w", err)
	}
	if err != nil {
		return nil, nil, stagesReconcilerCfg,
			fmt.Errorf("error starting cluster: %w", err)
	}

	return mgr, localCluster.GetClient(), stagesReconcilerCfg, nil
}

func (o *controllerOptions) setupArgoCDManager(ctx context.Context) (manager.Manager, error) {
	if !o.ArgoCDEnabled {
		o.Logger.Info("Argo CD integration is disabled")
		return nil, nil
	}

	// If the env var is undefined, this will resolve to kubeconfig for the
	// cluster the controller is running in.
	//
	// It is typically defined if this controller is running somewhere other
	// than where the Argo CD resources live. Two examples of this would
	// involve topologies wherein Kargo controllers run EITHER sharded
	// across application clusters OR in a centralized management cluster,
	// but with Argo CD deployed to a different management cluster.
	restCfg, err := kubernetes.GetRestConfig(ctx, o.ArgoCDKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading REST config for Argo CD controller manager: %w", err)
	}
	kubernetes.ConfigureQPSBurst(ctx, restCfg, o.QPS, o.Burst)
	restCfg.ContentType = runtime.ContentTypeJSON

	argocdNamespace := libargocd.Namespace()

	// There's a chance there is only permission to interact with Argo CD
	// Application resources in a single namespace, so we will use that
	// namespace when attempting to determine if Argo CD CRDs are installed.
	var exists bool
	if exists, err = argoCDExists(ctx, restCfg, argocdNamespace); !exists || err != nil {
		// If we are unable to determine if Argo CD is installed, we will
		// return an error and fail to start the controller. Note this
		// will only happen if we get an inconclusive response from the API
		// server (e.g. due to network issues), and not if Argo CD is not
		// installed.
		if err != nil {
			return nil, fmt.Errorf("unable to determine if Argo CD is installed: %w", err)
		}
		o.Logger.Info(
			"Argo CD integration was enabled, but no Argo CD CRDs were found. " +
				"Proceeding without Argo CD integration.",
		)
		return nil, nil
	}

	o.Logger.Info("Argo CD integration is enabled")

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kubernetes core API to Argo CD controller manager scheme: %w",
			err,
		)
	}
	if err = argocd.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Argo CD API to Argo CD controller manager scheme: %w",
			err,
		)
	}
	cacheOpts := cache.Options{} // Watches all namespaces by default
	if o.ArgoCDNamespaceOnly {
		cacheOpts.DefaultNamespaces = map[string]cache.Config{
			argocdNamespace: {},
		}
	}

	return ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
			Cache: cacheOpts,
		},
	)
}

func (o *controllerOptions) setupReconcilers(
	ctx context.Context,
	kargoMgr, argocdMgr manager.Manager,
	credentialsDB credentials.Database,
	stagesReconcilerCfg stages.ReconcilerConfig,
) error {
	var argoCDClient client.Client
	if argocdMgr != nil {
		argoCDClient = argocdMgr.GetClient()
	}

	healthCheckers.Initialize(argoCDClient)

	sharedIndexer := indexer.NewSharedFieldIndexer(kargoMgr.GetFieldIndexer())

	if promotionsReconcilerCfg := promotions.ReconcilerConfigFromEnv(); promotionsReconcilerCfg.Enable {
		if err := promotions.SetupReconcilerWithManager(
			ctx,
			kargoMgr,
			argocdMgr,
			promotion.NewLocalEngine(
				kargoMgr.GetClient(),
				argoCDClient,
				credentialsDB,
				promotion.DefaultExprDataCacheFn,
			),
			promotionsReconcilerCfg,
		); err != nil {
			return fmt.Errorf("error setting up Promotions reconciler: %w", err)
		}
	}

	if err := stages.NewRegularStageReconciler(
		stagesReconcilerCfg,
		health.NewAggregatingChecker(),
	).SetupWithManager(
		ctx,
		kargoMgr,
		argocdMgr,
		sharedIndexer,
	); err != nil {
		return fmt.Errorf("error setting up regular Stages reconciler: %w", err)
	}

	if err := stages.NewControlFlowStageReconciler(stagesReconcilerCfg).SetupWithManager(
		ctx,
		kargoMgr,
		sharedIndexer,
	); err != nil {
		return fmt.Errorf("error setting up control flow Stages reconciler: %w", err)
	}

	if err := warehouses.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		credentialsDB,
		subscription.DefaultSubscriberRegistry,
		warehouses.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up Warehouses reconciler: %w", err)
	}

	return nil
}

func (o *controllerOptions) startManagers(ctx context.Context, kargoMgr, argocdMgr manager.Manager) error {
	var (
		errChan = make(chan error)
		wg      = sync.WaitGroup{}
	)

	if argocdMgr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := argocdMgr.Start(ctx); err != nil {
				errChan <- fmt.Errorf("error starting Argo CD manager: %w", err)
				return
			}
			o.Logger.Debug("Argo CD manager started successfully")

			if !argocdMgr.GetCache().WaitForCacheSync(ctx) {
				errChan <- fmt.Errorf("failed to wait for Argo CD cache to sync")
				return
			}
			o.Logger.Debug("Argo CD cache synced successfully")
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := kargoMgr.Start(ctx); err != nil {
			errChan <- fmt.Errorf("error starting Kargo manager: %w", err)
			return
		}
		o.Logger.Debug("Kargo manager started successfully")

		if !kargoMgr.GetCache().WaitForCacheSync(ctx) {
			errChan <- fmt.Errorf("failed to wait for Kargo cache to sync")
			return
		}
		o.Logger.Debug("Kargo cache synced successfully")
	}()

	// Adapt wg to a channel that can be used in a select
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case err := <-errChan:
		return err
	case <-doneCh:
		return nil
	}
}
