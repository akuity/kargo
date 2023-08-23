package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/garbage"
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
				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(err, "error adding Kubernetes core API to scheme")
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return errors.Wrap(err, "error adding Kargo API to scheme")
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
