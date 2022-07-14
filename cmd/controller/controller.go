package controller

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/common/kubernetes"
	"github.com/akuityio/k8sta/internal/common/version"
	"github.com/akuityio/k8sta/internal/controller"
	"github.com/akuityio/k8sta/internal/scratch"
)

// RunController configures and runs the K8sTA controller.
func RunController(ctx context.Context) error {
	log.WithFields(log.Fields{
		"version": version.Version(),
		"commit":  version.Commit(),
	}).Info("Starting K8sTA Controller")

	config, err := scratch.K8staConfig()
	if err != nil {
		return errors.Wrap(err, "error reading K8sTA configuration")
	}

	kubeClient, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "error obtaining Kubernetes client")
	}

	argocdClient, err := argocdClient()
	if err != nil {
		return errors.Wrap(err, "error obtaining Argo CD client")
	}

	return controller.NewController(config, kubeClient, argocdClient).Run(ctx)
}
