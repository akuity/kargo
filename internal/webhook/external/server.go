package external

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

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

	// TODO: Register handlers here
	// mux.Handle("/", ...)

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
