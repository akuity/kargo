package main

import (
	"context"
	"errors"
	"fmt"
	stdruntime "runtime"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/garbage"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

type garbageCollectorOptions struct {
	KubeConfig string

	PprofBindAddress string

	Logger *logging.Logger
}

func newGarbageCollectorCommand() *cobra.Command {
	cmdOpts := &garbageCollectorOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
	}

	cmd := &cobra.Command{
		Use:               "garbage-collector",
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

func (o *garbageCollectorOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
	o.PprofBindAddress = os.GetEnv("PPROF_BIND_ADDRESS", "")
}

func (o *garbageCollectorOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()

	o.Logger.Info(
		"Starting Kargo Garbage Collector",
		"version", version.Version,
		"commit", version.GitCommit,
		"GOMAXPROCS", stdruntime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)

	mgr, err := o.setupManager(ctx)
	if err != nil {
		return fmt.Errorf("error setting up controller manager: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(fmt.Errorf("start manager: %w", err))
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		return errors.New("error waiting for cache sync")
	}

	cfg := garbage.CollectorConfigFromEnv()
	return garbage.NewCollector(mgr.GetClient(), cfg).Run(ctx)
}

func (o *garbageCollectorOptions) setupManager(ctx context.Context) (manager.Manager, error) {
	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("error loading REST config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding Kubernetes core API to scheme: %w", err)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding Kargo API to scheme: %w", err)
	}

	mgr, err := ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
			PprofBindAddress: o.PprofBindAddress,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing controller manager: %w", err)
	}

	// Index Promotions by Stage
	if err = mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Promotion{},
		indexer.PromotionsByStageField,
		indexer.PromotionsByStage,
	); err != nil {
		return nil, fmt.Errorf("error indexing Promotions by Stage: %w", err)
	}

	// Index Freight by Warehouse
	if err = mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByWarehouseField,
		indexer.FreightByWarehouse,
	); err != nil {
		return nil, fmt.Errorf("error indexing Freight by Warehouse: %w", err)
	}

	// Index Stages by Freight
	if err = mgr.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Stage{},
		indexer.StagesByFreightField,
		indexer.StagesByFreight,
	); err != nil {
		return nil, fmt.Errorf("error indexing Stages by Freight: %w", err)
	}
	return mgr, nil
}
