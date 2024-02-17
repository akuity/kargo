package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/controller/management/namespaces"
	"github.com/akuity/kargo/internal/controller/management/projects"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

func newManagementControllerCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "management-controller",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()

			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo Management Controller")

			var kargoMgr manager.Manager
			{
				restCfg, err :=
					kubernetes.GetRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
				if err != nil {
					return errors.Wrap(
						err,
						"error loading REST config for Kargo controller manager",
					)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes core API to Kargo controller manager "+
							"scheme",
					)
				}
				if err = rbacv1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes RBAC API to Kargo controller manager "+
							"scheme",
					)
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kargo API to Kargo controller manager scheme",
					)
				}
				if kargoMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme: scheme,
						Metrics: server.Options{
							BindAddress: "0",
						},
					},
				); err != nil {
					return errors.Wrap(err, "error initializing Kargo controller manager")
				}
			}

			if err := namespaces.SetupReconcilerWithManager(kargoMgr); err != nil {
				return errors.Wrap(err, "error setting up Namespaces reconciler")
			}

			if err := projects.SetupReconcilerWithManager(
				kargoMgr,
				projects.ReconcilerConfigFromEnv(),
			); err != nil {
				return errors.Wrap(err, "error setting up Projects reconciler")
			}

			return errors.Wrap(kargoMgr.Start(ctx), "error starting kargo manager")
		},
	}
}
