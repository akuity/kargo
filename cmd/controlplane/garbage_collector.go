package main

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/garbage"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

func newGarbageCollectorCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "garbage-collector",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo Garbage Collector")

			cfg := garbage.CollectorConfigFromEnv()

			var mgr manager.Manager
			{
				restCfg, err :=
					kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
				if err != nil {
					return errors.Wrap(err, "error loading REST config")
				}
				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(err, "error adding Kubernetes core API to scheme")
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return errors.Wrap(err, "error adding Kargo API to scheme")
				}
				if mgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme: scheme,
						Metrics: server.Options{
							BindAddress: "0",
						},
					},
				); err != nil {
					return errors.Wrap(err, "error initializing controller manager")
				}
				// Index Freight by Warehouse
				if err = kubeclient.IndexFreightByWarehouse(ctx, mgr); err != nil {
					return errors.Wrap(err, "error indexing Freight by Warehouse")
				}
				// Index Stages by Freight
				if err = kubeclient.IndexStagesByFreight(ctx, mgr); err != nil {
					return errors.Wrap(err, "error indexing Stages by Freight")
				}
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			go func() {
				if err := mgr.Start(ctx); err != nil {
					panic(errors.Wrap(err, "start manager"))
				}
			}()

			if !mgr.GetCache().WaitForCacheSync(ctx) {
				return errors.New("error waiting for cache sync")
			}

			return garbage.NewCollector(mgr.GetClient(), cfg).Run(ctx)
		},
	}
}
