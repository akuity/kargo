package main

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	libCluster "sigs.k8s.io/controller-runtime/pkg/cluster"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/types"
	"github.com/akuity/kargo/internal/webhook/external"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type externalWebhooksServerOptions struct {
	KubeConfig string
	QPS        float32
	Burst      int

	BindAddress string
	Port        string

	Logger *logging.Logger
}

func newExternalWebhooksServerCommand() *cobra.Command {
	cmdOpts := &externalWebhooksServerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
	}

	cmd := &cobra.Command{
		Use:               "external-webhooks-server",
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

func (o *externalWebhooksServerOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
	o.QPS = types.MustParseFloat32(os.GetEnv("KUBE_API_QPS", "50.0"))
	o.Burst = types.MustParseInt(os.GetEnv("KUBE_API_BURST", "300"))

	o.BindAddress = os.GetEnv("BIND_ADDRESS", "0.0.0.0")
	o.Port = os.GetEnv("PORT", "8080")
}

func (o *externalWebhooksServerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()
	o.Logger.Info(
		"Starting Kargo External Webhooks Server",
		"version", version.Version,
		"commit", version.GitCommit,
		"GOMAXPROCS", runtime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)

	serverCfg := external.ServerConfigFromEnv()

	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes client REST config: %w", err)
	}
	kubernetes.ConfigureQPSBurst(ctx, restCfg, o.QPS, o.Burst)

	cluster, err := libCluster.New(
		restCfg,
		func(clusterOptions *libCluster.Options) {
			clusterOptions.Client = client.Options{
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						&corev1.Secret{},
					},
				},
			}
		},
	)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	if err = kargoapi.AddToScheme(cluster.GetClient().Scheme()); err != nil {
		return fmt.Errorf("error adding Kargo API to scheme: %w", err)
	}

	err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.Warehouse{},
		indexer.WarehousesBySubscribedURLsField,
		indexer.WarehousesBySubscribedURLs,
	)
	if err != nil {
		return fmt.Errorf("error registering warehouse by repo url indexer: %w", err)
	}

	err = cluster.GetFieldIndexer().IndexField(
		ctx,
		&kargoapi.ProjectConfig{},
		indexer.ProjectConfigsByWebhookReceiverPathsField,
		indexer.ProjectConfigsByWebhookReceiverPaths,
	)
	if err != nil {
		return fmt.Errorf("error registering project configs by webhook receiver path indexer: %w", err)
	}

	go func() {
		err = cluster.Start(ctx)
	}()
	if !cluster.GetCache().WaitForCacheSync(ctx) {
		return fmt.Errorf("error waiting for cache to sync: %w", err)
	}
	if err != nil {
		return fmt.Errorf("error starting cluster: %w", err)
	}

	srv := external.NewServer(serverCfg, cluster.GetClient())
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", o.BindAddress, o.Port))
	if err != nil {
		return fmt.Errorf("error creating listener: %w", err)
	}
	defer l.Close()

	if err = srv.Serve(ctx, l); err != nil {
		return fmt.Errorf("error serving API: %w", err)
	}
	return nil
}
