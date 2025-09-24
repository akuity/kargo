package external

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/logging"
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
	// Health check endpoint. Keep health handling separate from the route handler.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// All other requests are delegated to the route handler.
	mux.HandleFunc("/", s.route)

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
