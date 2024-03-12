package main

import (
	"fmt"

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
					return fmt.Errorf("error loading REST config for Kargo controller manager: %w", err)
				}
				restCfg.ContentType = runtime.ContentTypeJSON

				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return fmt.Errorf(
						"error adding Kubernetes core API to Kargo controller manager scheme: %w",
						err,
					)
				}
				if err = rbacv1.AddToScheme(scheme); err != nil {
					return fmt.Errorf(
						"error adding Kubernetes RBAC API to Kargo controller manager scheme: %w",
						err,
					)
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return fmt.Errorf(
						"error adding Kargo API to Kargo controller manager scheme: %w",
						err,
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
					return fmt.Errorf("error initializing Kargo controller manager: %w", err)
				}
			}

			if err := namespaces.SetupReconcilerWithManager(kargoMgr); err != nil {
				return fmt.Errorf("error setting up Namespaces reconciler: %w", err)
			}

			if err := projects.SetupReconcilerWithManager(
				kargoMgr,
				projects.ReconcilerConfigFromEnv(),
			); err != nil {
				return fmt.Errorf("error setting up Projects reconciler: %w", err)
			}

			if err := kargoMgr.Start(ctx); err != nil {
				return fmt.Errorf("error starting kargo manager: %w", err)
			}
			return nil
		},
	}
}
