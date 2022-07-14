package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/common/file"
)

// ServerConfig represents optional configuration for an HTTP/S server.
type ServerConfig struct {
	// Port specifies the port the server should bind to / listen on.
	Port int
	// TLSEnabled specifies whether the server should serve HTTP (false) or HTTPS
	// (true).
	TLSEnabled bool
	// TLSCertPath is the path to a PEM-encoded x509 certificate that can be used
	// for serving HTTPS.
	TLSCertPath string
	// TLSKeyPath is the path to a PEM-encoded x509 private key that can be used
	// for serving HTTPS.
	TLSKeyPath string
}

// Server is an interface for an HTTP/S server. This is an improvement over the
// HTTP/S server built into Go's http package, as it exposes simple
// configuration options and a context-sensitive ListenAndServe function.
type Server interface {
	// ListenAndServe runs the HTTP/S server until the provided context is
	// canceled. This function always returns a non-nil error.
	ListenAndServe(ctx context.Context) error
}

// server
type server struct {
	config  ServerConfig
	handler http.Handler
}

// NewServer returns a new HTTP/S server.
func NewServer(handler http.Handler, config *ServerConfig) Server {
	if config == nil {
		config = &ServerConfig{}
	}
	if config.Port == 0 {
		config.Port = 8080
	}
	return &server{
		config:  *config,
		handler: handler,
	}
}

func (s *server) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.handler,
	}

	errCh := make(chan error)

	if s.config.TLSEnabled {
		if s.config.TLSCertPath == "" {
			return errors.New(
				"TLS was enabled, but no certificate path was specified",
			)
		}

		if s.config.TLSKeyPath == "" {
			return errors.New(
				"TLS was enabled, but no key path was specified",
			)
		}

		ok, err := file.Exists(s.config.TLSCertPath)
		if err != nil {
			return errors.Wrap(err, "error checking for existence of TLS cert")
		}
		if !ok {
			return errors.Errorf(
				"no TLS certificate found at path %s",
				s.config.TLSCertPath,
			)
		}

		if ok, err = file.Exists(s.config.TLSKeyPath); err != nil {
			return errors.Wrap(err, "error checking for existence of TLS key")
		}
		if !ok {
			return errors.Errorf("no TLS key found at path %s", s.config.TLSKeyPath)
		}

		log.Info("Server is listening")
		go func() {
			err := srv.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}()
	} else {
		log.Info("Server is listening")
		go func() {
			err := srv.ListenAndServe()
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Five second grace period on shutdown
		shutdownCtx, cancel :=
			context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx) // nolint: errcheck
		return ctx.Err()
	}
}
