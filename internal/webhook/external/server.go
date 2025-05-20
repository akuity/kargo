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

	mux.Handle("POST /", http.HandlerFunc(s.refreshWarehouseWebhook))
	// TODO(fuskovic): support additional providers

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

func (s *server) refreshWarehouseWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := (r.Context())
	logger := logging.LoggerFromContext(ctx)
	var projectConfigs kargoapi.ProjectConfigList
	err := s.client.List(
		ctx,
		&projectConfigs,
		client.MatchingFields{
			indexer.ProjectConfigsByWebhookReceiverPathsField: r.URL.Path,
		},
	)
	if err != nil {
		logger.Error(err, "failed to list project config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(projectConfigs.Items) == 0 {
		logger.Error(err, "no project configs found")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// There will always be only one project config
	// because two project configs can never have the same
	// receiver path because the receiver path is
	// a hash of the project name, provider and secret.
	pc := projectConfigs.Items[0]

	wrc, err := s.getWebhookReceiverConfig(
		r.URL.Path,
		pc,
	)
	if err != nil {
		logger.Error(err, "failed to get receiver config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			xhttp.WriteErrorJSON(w, xhttp.Error(err, http.StatusInternalServerError))
			return
		}
		token, ok := secret.StringData["token"]
		if !ok {
			logger.Error(err, "failed to get github token from secret")
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					errors.New("failed to get github token"),
					http.StatusInternalServerError,
				),
			)
			return
		}
		githubHandler(s.client, token)(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *server) getWebhookReceiverConfig(
	receiverPath string,
	pc kargoapi.ProjectConfig,
) (*kargoapi.WebhookReceiverConfig, error) {
	var whrc *kargoapi.WebhookReceiverConfig
	for _, config := range pc.Spec.WebhookReceiverConfigs { // nolint: nilness, lll // impossible for project config spec to be empty
		var configType string
		if config.GitHub != nil {
			configType = kargoapi.WebhookReceiverTypeGitHub
		}
		target := GenerateWebhookPath(
			pc.Name,
			configType,
			config.GitHub.SecretRef.Name,
		)
		if receiverPath == target {
			whrc = &config
			break
		}
	}
	if whrc == nil {
		return nil, fmt.Errorf(
			"failed to find receiver config with path %q in project config %q",
			receiverPath,
			pc.Name,
		)
	}
	return whrc, nil
}
