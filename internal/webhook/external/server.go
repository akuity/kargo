package external

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

type server struct {
	cfg    ServerConfig
	client client.Client
}

type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

func NewServer(cfg ServerConfig, cl client.Client) Server {
	return &server{
		cfg:    cfg,
		client: cl,
	}
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	logger := logging.LoggerFromContext(ctx)
	srv := &http.Server{
		Handler:           http.HandlerFunc(s.route),
		ReadHeaderTimeout: time.Minute,
	}
	errCh := make(chan error)
	go func() {
		if s.cfg.TLSConfig != nil {
			errCh <- srv.ServeTLS(
				l,
				s.cfg.TLSConfig.CertPath,
				s.cfg.TLSConfig.KeyPath,
			)
		} else {
			errCh <- srv.Serve(l)
		}
	}()

	logger.Info(
		"Server is listening",
		"tls", s.cfg.TLSConfig != nil,
		"address", l.Addr().String(),
	)

	select {
	case <-ctx.Done():
		logger.Info("Gracefully stopping server...")
		time.Sleep(s.cfg.GracefulShutdownTimeout)
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// route retrieves the project configurations that match the request path and
// determines the appropriate project + webhook receiver configuration to use.
// If a matching project configuration is found, it calls the appropriate
// handler based on the type of webhook receiver configured (e.g., GitHub).
// If no matching project configuration or webhook receiver is found, it returns
// a 404 Not Found error.
func (s *server) route(w http.ResponseWriter, r *http.Request) {
	ctx := (r.Context())
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("refresh route handler called", "path", r.URL.Path)
	var projectConfigs kargoapi.ProjectConfigList
	err := s.client.List(
		ctx,
		&projectConfigs,
		client.MatchingFields{
			indexer.ProjectConfigsByWebhookReceiverPathsField: r.URL.Path,
		},
	)
	if err != nil {
		logger.Error(err, "failed to list project configs")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(projectConfigs.Items) == 0 {
		logger.Info("no project configs found")
		http.Error(w, "no project configs found for the request", http.StatusNotFound)
		return
	}

	// Projects are allowed to have, at most, a single ProjectConfig
	pc := projectConfigs.Items[0]
	logger.Info("found project config", "project-config", pc)
	wrc, err := s.getWebhookReceiverConfig(pc, r.URL.Path)
	if err != nil {
		logger.Error(err, "failed to find webhook receiver config")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	logger.Debug("webhook receiver config found", "webhook-receiver-config", wrc)
	switch {
	case wrc.GitHub != nil:
		githubHandler(
			s.client,
			pc.Namespace,
			wrc.GitHub.SecretRef.Name,
		)(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *server) getWebhookReceiverConfig(
	pc kargoapi.ProjectConfig,
	receiverPath string,
) (*kargoapi.WebhookReceiverConfig, error) {
	var whrc *kargoapi.WebhookReceiverConfig
	for _, r := range pc.Status.WebhookReceivers {
		if receiverPath == r.Path {
			for _, cfg := range pc.Spec.WebhookReceivers {
				if cfg.Name == r.Name {
					whrc = &cfg
					break
				}
			}
			break
		}
	}
	if whrc == nil {
		return nil, fmt.Errorf(
			"failed to find webhook receiver config with path %q in project config %q",
			receiverPath,
			pc.Name,
		)
	}
	return whrc, nil
}
