package main

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"github.com/spf13/cobra"
	libCluster "sigs.k8s.io/controller-runtime/pkg/cluster"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/webhook/external"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type externalWebhooksServerOptions struct {
	KubeConfig string

	Host string
	Port string

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
	o.Host = os.GetEnv("HOST", "0.0.0.0")
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

	cluster, err := libCluster.New(restCfg)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	if err = kargoapi.AddToScheme(cluster.GetClient().Scheme()); err != nil {
		return fmt.Errorf("error adding Kargo API to Kargo manager scheme: %w", err)
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

	srv := external.NewServer(serverCfg, cluster.GetClient())
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", o.Host, o.Port))
	if err != nil {
		return fmt.Errorf("error creating listener: %w", err)
	}
	defer l.Close()

	if err = srv.Serve(ctx, l); err != nil {
		return fmt.Errorf("error serving API: %w", err)
	}
	return nil
}
