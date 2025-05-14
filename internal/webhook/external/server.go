package external

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/types"
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
	var projects kargoapi.ProjectList
	err := s.client.List(
		ctx,
		&projects,
		client.MatchingFields{
			indexer.ProjectsByWebhookReceiverPathsField: r.URL.Path,
		},
	)
	if err != nil {
		logger.Error(err, "failed to list projects")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(projects.Items) == 0 {
		logger.Error(err, "no projects found")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// There will always be only one project
	// because two projects can never have the same
	// receiver path because the receiver path is
	// a hash of the project name, provider and secret.
	project := projects.Items[0]

	config, err := s.getReceiverConfig(ctx, r.URL.Path, project)
	if err != nil {
		logger.Error(err, "failed to get receiver config")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Debug("receiver config found", "config", config)
	switch config.Type {
	case kargoapi.WebhookReceiverTypeGitHub:
		githubHandler(s.client, config.SecretRef)(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *server) getReceiverConfig(
	ctx context.Context,
	receiverPath string,
	project kargoapi.Project,
) (*kargoapi.WebhookReceiverConfig, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"receiverPath", receiverPath,
		"projectName", project.Name,
		"projectNamespace", project.Namespace,
	)

	receiver, err := s.getReceiver(receiverPath, project)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get receiver for project %q in namespace %q: %w",
			project.Name,
			project.Namespace,
			err,
		)
	}

	logger.Debug("getting project config")
	var projectConfig *kargoapi.ProjectConfig
	err = s.client.Get(ctx,
		types.NamespacedName{
			Name:      project.Name,
			Namespace: project.Namespace,
		},
		projectConfig,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get project config for project %q in namespace %q: %w",
			project.Name,
			project.Namespace,
			err,
		)
	}

	var whrc *kargoapi.WebhookReceiverConfig
	for _, config := range projectConfig.Spec.WebhookReceiverConfigs { // nolint: nilness, lll // impossible for project config spec to be empty
		target := GenerateWebhookPath(
			project.Name,
			config.Type,
			config.SecretRef,
		)
		if receiver.Path == target {
			whrc = &config
		}
	}
	if whrc == nil {
		return nil, fmt.Errorf(
			"failed to find receiver config with path %q in project %q",
			receiver.Path,
			project.Name,
		)
	}
	return whrc, nil
}

func (s *server) getReceiver(
	receiverPath string,
	project kargoapi.Project,
) (*kargoapi.WebhookReceiver, error) {
	var receiver *kargoapi.WebhookReceiver
	for _, r := range project.Status.WebhookReceivers {
		if r.Path == receiverPath {
			receiver = &r
		}
	}
	if receiver == nil {
		return nil, fmt.Errorf(
			"failed to find receiver with path %q in project %q",
			receiverPath,
			project.Name,
		)
	}
	return receiver, nil
}
