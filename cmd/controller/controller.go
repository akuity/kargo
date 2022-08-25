package controller

import (
	"context"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/argoproj/argo-cd/v2/util/settings"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/common/kubernetes"
	"github.com/akuityio/k8sta/internal/common/version"
	"github.com/akuityio/k8sta/internal/controller"
)

// RunController configures and runs the K8sTA controller.
func RunController(ctx context.Context, config config.Config) error {
	log.WithFields(log.Fields{
		"version": version.Version(),
		"commit":  version.Commit(),
	}).Info("Starting K8sTA Controller")

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
		return errors.Wrap(err, "error adding K8sTA API to scheme")
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
		settings.NewSettingsManager(ctx, kubeClient, ""),
		kubeClient,
	)

	if err := controller.SetupTrackReconcilerWithManager(
		ctx,
		config,
		mgr,
		argoDB,
	); err != nil {
		return errors.Wrap(err, "error setting up Track reconciler")
	}

	if err := controller.SetupTicketReconcilerWithManager(
		ctx,
		config,
		mgr,
		argoDB,
	); err != nil {
		return errors.Wrap(err, "error setting up Ticket reconciler")
	}

	return errors.Wrap(
		mgr.Start(ctrl.SetupSignalHandler()),
		"error running manager",
	)
}
