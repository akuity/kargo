package main

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/promotions"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/stages"
	"github.com/akuity/kargo/internal/controller/warehouses"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/types"
	versionpkg "github.com/akuity/kargo/internal/version"

	_ "github.com/akuity/kargo/internal/gitprovider/github"
)

type controllerOptions struct {
	ShardName  string
	KubeConfig string

	ArgoCDEnabled       bool
	ArgoCDKubeConfig    string
	ArgoCDNamespaceOnly bool

	RolloutsEnabled      bool
	RolloutsKubeConfig   string
	AnalysisRunNamespace string

	Logger *log.Logger
}

func newControllerCommand() *cobra.Command {
	cmdOpts := &controllerOptions{
		Logger: log.StandardLogger(),
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
	o.ArgoCDEnabled = types.MustParseBool(os.GetEnv("ARGOCD_INTEGRATION_ENABLED", "true"))
	o.ArgoCDKubeConfig = os.GetEnv("ARGOCD_KUBECONFIG", "")
	o.ArgoCDNamespaceOnly = types.MustParseBool(os.GetEnv("ARGOCD_WATCH_ARGOCD_NAMESPACE_ONLY", "false"))
	o.RolloutsEnabled = types.MustParseBool(os.GetEnv("ROLLOUTS_INTEGRATION_ENABLED", "true"))
	o.RolloutsKubeConfig = os.GetEnv("ROLLOUTS_KUBECONFIG", "")
	o.AnalysisRunNamespace = os.GetEnv("ROLLOUTS_ANALYSIS_RUNS_NAMESPACE", "")
}

func (o *controllerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()

	startupLogEntry := o.Logger.WithFields(log.Fields{
		"version": version.Version,
		"commit":  version.GitCommit,
	})
	if o.ShardName != "" {
		startupLogEntry = startupLogEntry.WithField("shard", o.ShardName)
	}
	startupLogEntry.Info("Starting Kargo Controller")

	kargoMgr, err := o.setupKargoManager(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Kargo controller manager: %w", err)
	}

	argocdMgr, err := o.setupArgoCDManager(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Argo CD Application controller manager: %w", err)
	}

	rolloutsMgr, err := o.setupRolloutsManager(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Argo Rollouts AnalysisRun controller manager: %w", err)
	}

	credentialsDB := credentials.NewKubernetesDatabase(
		kargoMgr.GetClient(),
		credentials.KubernetesDatabaseConfigFromEnv(),
	)

	if err := o.setupReconcilers(ctx, kargoMgr, argocdMgr, rolloutsMgr, credentialsDB); err != nil {
		return fmt.Errorf("error setting up reconcilers: %w", err)
	}

	return o.startManagers(ctx, kargoMgr, argocdMgr, rolloutsMgr)
}

func (o *controllerOptions) setupKargoManager(ctx context.Context) (manager.Manager, error) {
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
		return nil, fmt.Errorf("error loading REST config for Kargo controller manager: %w", err)
	}
	restCfg.ContentType = runtime.ContentTypeJSON

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kubernetes core API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = rollouts.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Argo Rollouts API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kargo API to Kargo controller manager scheme: %w",
			err,
		)
	}

	secretReq, err := controller.GetCredentialsRequirement()
	if err != nil {
		return nil, fmt.Errorf("error getting label requirement for credentials Secrets: %w", err)
	}

	cacheOpts := cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			// Only watch Secrets matching the label requirements
			// for credentials.
			&corev1.Secret{}: {
				Label: labels.NewSelector().Add(*secretReq),
			},
		},
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
	if !argoCDExists(ctx, restCfg, argocdNamespace) {
		o.Logger.Warn(
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

func (o *controllerOptions) setupRolloutsManager(ctx context.Context) (manager.Manager, error) {
	if !o.RolloutsEnabled {
		o.Logger.Info("Argo Rollouts integration is disabled")
		return nil, nil
	}

	// If the env var is undefined, this will resolve to kubeconfig for the
	// cluster the controller is running in.
	//
	// It is typically defined if this controller is running somewhere other
	// than a cluster suitable for executing Argo Rollouts AnalysesRuns and
	// user-defined workloads. An example of this would be a topology
	// wherein the Kargo and Argo CD controllers both run in management
	// clusters, that are a not suitable for this purpose.
	restCfg, err := kubernetes.GetRestConfig(ctx, o.RolloutsKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading REST config for Argo Rollouts controller manager: %w", err)
	}
	restCfg.ContentType = runtime.ContentTypeJSON

	if !argoRolloutsExists(ctx, restCfg) {
		o.Logger.Warn(
			"Argo Rollouts integration was enabled, but no Argo Rollouts CRDs were found. " +
				"Proceeding without Argo Rollouts integration.",
		)
		return nil, nil
	}

	o.Logger.Info("Argo Rollouts integration is enabled")

	scheme := runtime.NewScheme()
	if err = rollouts.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Argo Rollouts API to Argo Rollouts controller manager scheme: %w",
			err,
		)
	}

	cacheOpts := cache.Options{} // Watches all namespaces by default
	if o.AnalysisRunNamespace != "" {
		// TODO: When NOT sharded, Kargo can simply create AnalysisRun
		// resources in the project namespaces. When sharded, AnalysisRun
		// resources must be created IN the shard clusters (not the Kargo
		// control plane cluster) and project namespaces do not exist in the
		// shard clusters. We need a place to put them, so for now we allow
		// the user to specify a namespace that that exists on each shard for
		// this purpose. Note that the namespace does not need to be the same
		// on every shard. This may be one of the weaker points in our tenancy
		// model and can stand to be improved.
		cacheOpts.DefaultNamespaces = map[string]cache.Config{
			o.AnalysisRunNamespace: {},
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
	kargoMgr, argocdMgr, rolloutsMgr manager.Manager,
	credentialsDB credentials.Database,
) error {
	if err := promotions.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		argocdMgr,
		credentialsDB,
		o.ShardName,
	); err != nil {
		return fmt.Errorf("error setting up Promotions reconciler: %w", err)
	}

	if err := stages.SetupReconcilerWithManager(
		ctx,
		kargoMgr,
		argocdMgr,
		rolloutsMgr,
		stages.ReconcilerConfigFromEnv(),
	); err != nil {
		return fmt.Errorf("error setting up Stages reconciler: %w", err)
	}

	if err := warehouses.SetupReconcilerWithManager(
		kargoMgr,
		credentialsDB,
		o.ShardName,
	); err != nil {
		return fmt.Errorf("error setting up Warehouses reconciler: %w", err)
	}

	return nil
}

func (o *controllerOptions) startManagers(ctx context.Context, kargoMgr, argocdMgr, rolloutsMgr manager.Manager) error {
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

	if rolloutsMgr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := rolloutsMgr.Start(ctx); err != nil {
				errChan <- fmt.Errorf("error starting rollouts manager: %w", err)
			}
		}()
	}

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
