package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	"github.com/akuityio/k8sta/internal/common/client"
	"github.com/akuityio/k8sta/internal/common/config"
	libHTTP "github.com/akuityio/k8sta/internal/common/http"
	"github.com/akuityio/k8sta/internal/common/version"
	"github.com/akuityio/k8sta/internal/dockerhub"
)

// RunServer configures and runs the K8sTA Server.
func RunServer(ctx context.Context, config config.Config) error {
	version := version.GetVersion()

	serverConfig, err := serverConfig()
	if err != nil {
		return errors.Wrap(err, "error reading server configuration")
	}

	var tlsStatus = "disabled"
	if serverConfig.TLSEnabled {
		tlsStatus = "enabled"
	}
	log.WithFields(log.Fields{
		"version": version.Version,
		"commit":  version.GitCommit,
		"port":    serverConfig.Port,
		"tls":     tlsStatus,
	}).Info("Starting K8sTA Server")

	scheme := runtime.NewScheme()
	if err = api.AddToScheme(scheme); err != nil {
		return errors.Wrap(err, "error adding K8sTA API to scheme")
	}
	controllerRuntimeClient, err := client.New(scheme)
	if err != nil {
		return errors.Wrap(err, "error obtaining controller runtime client")
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
			dockerhub.NewService(config, controllerRuntimeClient),
		)
		if err != nil {
			return errors.Wrap(err, "error creating handler for Docker Hub webhooks")
		}
		dockerhubWebhookHandler = dockerhub.NewTokenFilter(
			filterConfig,
			handler.ServeHTTP,
		)
	}

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.Handle("/dockerhub", dockerhubWebhookHandler).Methods(http.MethodPost)
	// TODO: Support more triggers here!
	// - Container registries:
	//   - ghcr
	//   - gcr
	//   - acr
	// - Other?
	router.HandleFunc("/healthz", libHTTP.Healthz).Methods(http.MethodGet)

	return libHTTP.NewServer(
		router,
		&serverConfig,
	).ListenAndServe(ctx)
}
