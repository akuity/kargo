package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/kargo/internal/common/config"
	libHTTP "github.com/akuityio/kargo/internal/common/http"
	"github.com/akuityio/kargo/internal/common/version"
)

// RunServer configures and runs the Kargo Server.
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
	}).Info("Starting Kargo Server")

	// TODO: We will probably want to uncomment this in the future, because it's
	// the key to giving any handlers we register below the ability to manipulate
	// the Kargo API.

	// scheme := runtime.NewScheme()
	// if err = api.AddToScheme(scheme); err != nil {
	// 	return errors.Wrap(err, "error adding Kargo API to scheme")
	// }
	// controllerRuntimeClient, err := client.New(scheme)
	// if err != nil {
	// 	return errors.Wrap(err, "error obtaining controller runtime client")
	// }

	router := mux.NewRouter()
	router.StrictSlash(true)
	// TODO: Since switching to polling for updated images (like Image Updater
	// does), we removed the only non-trivial endpoint that was here before -- the
	// one for handling inbound webhooks from Docker Hub -- but this is where we
	// can register new handlers for other things in the future.
	router.HandleFunc("/healthz", libHTTP.Healthz).Methods(http.MethodGet)

	return libHTTP.NewServer(
		router,
		&serverConfig,
	).ListenAndServe(ctx)
}
