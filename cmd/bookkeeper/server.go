package bookkeeper

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/bookkeeper"
	"github.com/akuityio/k8sta/internal/common/config"
	libHTTP "github.com/akuityio/k8sta/internal/common/http"
	"github.com/akuityio/k8sta/internal/common/version"
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
	}).Info("Starting Bookkeeper Server")

	router := mux.NewRouter()
	router.StrictSlash(true)
	router.Handle(
		"/v1alpha1",
		bookkeeper.NewHandler(
			config,
			bookkeeper.NewService(config),
		),
	).Methods(http.MethodPost)
	router.HandleFunc("/healthz", libHTTP.Healthz).Methods(http.MethodGet)

	return libHTTP.NewServer(
		router,
		&serverConfig,
	).ListenAndServe(ctx)
}
