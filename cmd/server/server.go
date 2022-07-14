package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	libHTTP "github.com/akuityio/k8sta/internal/common/http"
	"github.com/akuityio/k8sta/internal/common/kubernetes"
	"github.com/akuityio/k8sta/internal/common/signals"
	"github.com/akuityio/k8sta/internal/common/version"
	"github.com/akuityio/k8sta/internal/dockerhub"
	"github.com/akuityio/k8sta/internal/scratch"
)

// RunServer configures and runs the K8sTA Server.
func RunServer(ctx context.Context) error {
	serverConfig, err := serverConfig()
	if err != nil {
		return errors.Wrap(err, "error reading server configuration")
	}

	var tlsStatus = "disabled"
	if serverConfig.TLSEnabled {
		tlsStatus = "enabled"
	}
	log.WithFields(log.Fields{
		"version": version.Version(),
		"commit":  version.Commit(),
		"port":    serverConfig.Port,
		"tls":     tlsStatus,
	}).Info("Starting K8sTA Server")

	config, err := scratch.K8staConfig()
	if err != nil {
		return errors.Wrap(err, "error reading K8sTA configuration")
	}

	kubeClient, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "error obtaining Kubernetes client")
	}

	// Wire together everything for handling webhooks from Docker Hub...
	var dockerhubWebhookHandler http.Handler
	{
		filterConfig, err := dockerhubFilterConfig()
		if err != nil {
			return errors.Wrap(
				err,
				"error creating authentication filter for Docker Hub webhooks",
			)
		}
		handler, err := dockerhub.NewHandler(
			dockerhub.NewService(config, kubeClient),
		)
		if err != nil {
			return errors.Wrap(err, "error creating handler for Docker Hub webhooks")
		}
		dockerhubWebhookHandler = dockerhub.NewTokenFilter(
			filterConfig,
		).Decorate(handler.ServeHTTP)
	}

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.Handle("/dockerhub", dockerhubWebhookHandler).Methods(http.MethodPost)
	// TODO: Support more triggers here!
	router.HandleFunc("/healthz", libHTTP.Healthz).Methods(http.MethodGet)

	return libHTTP.NewServer(
		router,
		&serverConfig,
	).ListenAndServe(signals.Context())
}
