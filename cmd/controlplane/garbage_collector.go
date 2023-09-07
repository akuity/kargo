package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

			var kubeClient client.Client
			{
				restCfg, err :=
					kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
				if err != nil {
					return errors.Wrap(err, "error loading REST config")
				}
				scheme, err := kubeclient.NewGarbageCollectorScheme()
				if err != nil {
					return errors.Wrap(err, "new garbage collector scheme")
				}
				if kubeClient, err = client.New(
					restCfg,
					client.Options{
						Scheme: scheme,
					},
				); err != nil {
					return errors.Wrap(err, "error initializing Kubernetes client")
				}
			}

			return garbage.NewCollector(kubeClient, cfg).Run(ctx)
		},
	}
}
