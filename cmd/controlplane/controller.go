package main

import (
	"context"
	"fmt"
	stdruntime "runtime"
	"sync"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/promotions"
	"github.com/akuity/kargo/internal/controller/stages"
	"github.com/akuity/kargo/internal/controller/warehouses"
	"github.com/akuity/kargo/internal/credentials"
	credsdb "github.com/akuity/kargo/internal/credentials/kubernetes"
	"github.com/akuity/kargo/internal/health"
	healthCheckers "github.com/akuity/kargo/internal/health/checker/builtin"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/leaderelection"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/promotion"
	promotionStepRunners "github.com/akuity/kargo/internal/promotion/runner/builtin"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/types"
	pkgPromo "github.com/akuity/kargo/pkg/promotion"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type controllerOptions struct {
	ShardName  string
	KubeConfig string

	LeaderElectionEnabled         bool
	LeaderElectionReleaseOnCancel bool
	LeaderElectionLeaseDuration   *time.Duration
	LeaderElectionRenewDeadline   *time.Duration
	LeaderElectionRetryPeriod     *time.Duration

	ArgoCDEnabled       bool
	ArgoCDKubeConfig    string
	ArgoCDNamespaceOnly bool

	PprofBindAddress string

	Logger *logging.Logger
}

func newControllerCommand() *cobra.Command {
	cmdOpts := &controllerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
	}

	cmd := &cobra.Command{
		Use:               "controller",
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

func (o *controllerOptions) complete() {
	o.ShardName = os.GetEnv("SHARD_NAME", "")
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")

	o.LeaderElectionEnabled = types.MustParseBool(os.GetEnv("LEADER_ELECTION_ENABLED", "false"))
	o.LeaderElectionReleaseOnCancel = types.MustParseBool(os.GetEnv("LEADER_ELECTION_RELEASE_ON_CANCEL", ""))
	o.LeaderElectionLeaseDuration = types.MustParseDuration(os.GetEnv("LEADER_ELECTION_LEASE_DURATION", ""))
	o.LeaderElectionRenewDeadline = types.MustParseDuration(os.GetEnv("LEADER_ELECTION_RENEW_DEADLINE", ""))
	o.LeaderElectionRetryPeriod = types.MustParseDuration(os.GetEnv("LEADER_ELECTION_RETRY_PERIOD", ""))

	o.ArgoCDEnabled = types.MustParseBool(os.GetEnv("ARGOCD_INTEGRATION_ENABLED", "true"))
	o.ArgoCDKubeConfig = os.GetEnv("ARGOCD_KUBECONFIG", "")
	o.ArgoCDNamespaceOnly = types.MustParseBool(os.GetEnv("ARGOCD_WATCH_ARGOCD_NAMESPACE_ONLY", "false"))

	o.PprofBindAddress = os.GetEnv("PPROF_BIND_ADDRESS", "")
}

func (o *controllerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()

	startupLogger := o.Logger.WithValues(
		"version", version.Version,
		"commit", version.GitCommit,
		"GOMAXPROCS", stdruntime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)
	if o.ShardName != "" {
		startupLogger = startupLogger.WithValues("shard", o.ShardName)
	}
	startupLogger.Info("Starting Kargo Controller")

	kargoMgr, stagesReconcilerCfg, err := o.setupKargoManager(
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
		ctx,
		kargoMgr.GetClient(),
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
) (manager.Manager, stages.ReconcilerConfig, error) {
	// If the env var is undefined, this will resolve to kubeconfig for the
	// cluster the controller is running in.
	//
	// It is typically defined if this controller is running somewhere other
	// than where the Kargo resources live. One example of this would be a
	// sharded topology wherein Kargo controllers run on application
	// clusters, with Kargo resources hosted in a centralized management
	// cluster.
	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return nil, stagesReconcilerCfg,
			fmt.Errorf("error loading REST config for Kargo controller manager: %w", err)
	}
	restCfg.ContentType = runtime.ContentTypeJSON

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, stagesReconcilerCfg, fmt.Errorf(
			"error adding Kubernetes core API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, stagesReconcilerCfg, fmt.Errorf(
			"error adding Kargo API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if stagesReconcilerCfg.RolloutsIntegrationEnabled {
		var exists bool
		if exists, err = argoRolloutsExists(ctx, restCfg); exists {
			o.Logger.Info("Argo Rollouts integration is enabled")
			if err = rollouts.AddToScheme(scheme); err != nil {
				return nil, stagesReconcilerCfg, fmt.Errorf(
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
				return nil, stagesReconcilerCfg, fmt.Errorf(
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

	shardReq, err := controller.GetShardRequirement(stagesReconcilerCfg.ShardName)
	if err != nil {
		return nil, stagesReconcilerCfg, fmt.Errorf("error getting shard requirement: %w", err)
	}
	shardSelector := labels.NewSelector().Add(*shardReq)

	mgr, err := ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			Client: client.Options{
				Cache: &client.CacheOptions{
					// The controller does not have cluster-wide permissions, to
					// get/list/watch Secrets. Its access to Secrets grows and shrinks
					// dynamically as Projects are created and deleted. We disable caching
					// here since the underlying informer will not be able to watch
					// Secrets in all namespaces.
					DisableFor: []client.Object{&corev1.Secret{}},
				},
			},
			Cache: cache.Options{
				// When Kargo is sharded, we expect the controller to only handle
				// resources in the shard it is responsible for. This is enforced
				// by the following label selectors on the informers, EXCEPT for
				// Warehouses — which should be accessible by all controllers in
				// a sharded setup, but handled by only one controller at a time.
				ByObject: map[client.Object]cache.ByObject{
					&kargoapi.Stage{}:     {Label: shardSelector},
					&kargoapi.Promotion{}: {Label: shardSelector},
				},
			},
			LeaderElection:                o.LeaderElectionEnabled,
			LeaderElectionID:              leaderelection.GenerateIDFromLabelSelector("kargo-controller", shardSelector),
			LeaderElectionReleaseOnCancel: o.LeaderElectionReleaseOnCancel,
			LeaseDuration:                 o.LeaderElectionLeaseDuration,
			RenewDeadline:                 o.LeaderElectionRenewDeadline,
			Metrics: server.Options{
				BindAddress: "0",
			},
			PprofBindAddress: o.PprofBindAddress,
		},
	)
	return mgr, stagesReconcilerCfg, err
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

	promotionStepRunners.Initialize(kargoMgr.GetClient(), argoCDClient, credentialsDB)
	healthCheckers.Initialize(argoCDClient)

	sharedIndexer := indexer.NewSharedFieldIndexer(kargoMgr.GetFieldIndexer())

	if err := promotions.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		argocdMgr,
		promotion.NewSimpleEngine(kargoMgr.GetClient()),
		promotions.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up Promotions reconciler: %w", err)
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
				errChan <- fmt.Errorf("error starting argo cd manager: %w", err)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := kargoMgr.Start(ctx); err != nil {
			errChan <- fmt.Errorf("error starting kargo manager: %w", err)
		}
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

func RegisterPromotionStepRunner(runner pkgPromo.StepRunner) {
	promotion.RegisterStepRunner(runner)
}
