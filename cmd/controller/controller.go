package controller

import (
	"context"

	"github.com/akuityio/bookkeeper"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/common/config"
	"github.com/akuityio/kargo/internal/common/kubernetes"
	"github.com/akuityio/kargo/internal/common/version"
	"github.com/akuityio/kargo/internal/controller"
)

// RunController configures and runs the Kargo controller.
func RunController(ctx context.Context, config config.Config) error {
	version := version.GetVersion()

	log.WithFields(log.Fields{
		"version": version.Version,
		"commit":  version.GitCommit,
	}).Info("Starting Kargo Controller")

	mgrConfig, err := ctrl.GetConfig()
	if err != nil {
		return errors.Wrap(err, "error getting manager config")
	}
	scheme := runtime.NewScheme()
	if err = clientgoscheme.AddToScheme(scheme); err != nil {
		return errors.Wrap(err, "error adding Kubernetes API to scheme")
	}
	if err = argocd.AddToScheme(scheme); err != nil {
		return errors.Wrap(err, "error adding ArgoCD API to scheme")
	}
	if err = api.AddToScheme(scheme); err != nil {
		return errors.Wrap(err, "error adding Kargo API to scheme")
	}
	mgr, err := ctrl.NewManager(
		mgrConfig,
		ctrl.Options{
			Scheme: scheme,
			Port:   9443,
		},
	)
	if err != nil {
		return errors.Wrap(err, "error creating manager")
	}

	kubeClient, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "error obtaining Kubernetes client")
	}
	argoDB := db.NewDB(
		"",
		// TODO: Do not hard-code the namespace
		settings.NewSettingsManager(ctx, kubeClient, "argo-cd"),
		kubeClient,
	)

	if err := controller.SetupEnvironmentReconcilerWithManager(
		ctx,
		config,
		mgr,
		kubeClient,
		argoDB,
		bookkeeper.NewService(
			&bookkeeper.ServiceOptions{
				LogLevel: bookkeeper.LogLevel(config.LogLevel),
			},
		),
	); err != nil {
		return errors.Wrap(err, "error setting up Environment reconciler")
	}

	return errors.Wrap(
		mgr.Start(ctrl.SetupSignalHandler()),
		"error running manager",
	)
}
