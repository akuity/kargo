package main

import (
	"context"
	"fmt"

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
	"github.com/akuity/kargo/internal/controller/management/upgrade"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

type managementControllerOptions struct {
	KubeConfig string

	Logger *logging.Logger
}

func newManagementControllerCommand() *cobra.Command {
	cmdOpts := &managementControllerOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
	}

	cmd := &cobra.Command{
		Use:               "management-controller",
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

func (o *managementControllerOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
}

func (o *managementControllerOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()

	o.Logger.Info(
		"Starting Kargo Management Controller",
		"version", version.Version,
		"commit", version.GitCommit,
	)

	kargoMgr, err := o.setupManager(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Kargo controller manager: %w", err)
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

	if err := upgrade.SetupServiceAccountReconcilerWithManager(kargoMgr); err != nil {
		return fmt.Errorf("error setting up ServiceAccount reconciler: %w", err)
	}

	if err := kargoMgr.Start(ctx); err != nil {
		return fmt.Errorf("error starting kargo manager: %w", err)
	}
	return nil
}

func (o *managementControllerOptions) setupManager(ctx context.Context) (manager.Manager, error) {
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
	if err = rbacv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kubernetes RBAC API to Kargo controller manager scheme: %w",
			err,
		)
	}
	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf(
			"error adding Kargo API to Kargo controller manager scheme: %w",
			err,
		)
	}

	return ctrl.NewManager(
		restCfg,
		ctrl.Options{
			Scheme: scheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
		},
	)
}
