package external

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
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
	mux := http.NewServeMux()

	mux.Handle("POST /", http.HandlerFunc(s.refreshWarehouseHandler))

	srv := &http.Server{
		Handler:           mux,
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

func (s *server) refreshWarehouseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := (r.Context())
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("refresh warehouse handler called", "path", r.URL.Path)
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
		var secret corev1.Secret
		err := s.client.Get(ctx,
			client.ObjectKey{
				Name:      wrc.GitHub.SecretRef.Name,
				Namespace: pc.Namespace,
			},
			&secret,
		)
		if err != nil {
			logger.Error(err, "failed to get github secret")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("failed to get github secret %q: %w", wrc.GitHub.SecretRef.Name, err),
					http.StatusNotFound,
				),
			)
			return
		}
		token, ok := secret.Data["token"]
		if !ok {
			logger.Error(
				errors.New("failed to get github token from secret"),
				"no value for 'token' key",
			)
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("missing github token in secret"),
					http.StatusInternalServerError,
				),
			)
			return
		}
		githubHandler(s.client, pc.Namespace, string(token))(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *server) getWebhookReceiverConfig(
	pc kargoapi.ProjectConfig,
	receiverPath string,
) (*kargoapi.WebhookReceiverConfig, error) {
	var whrc *kargoapi.WebhookReceiverConfig
	for i, r := range pc.Status.WebhookReceivers {
		if receiverPath == r.Path {
			whrc = &pc.Spec.WebhookReceivers[i]
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
