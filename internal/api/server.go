package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/logging"
)

type server struct {
	cfg config.APIConfig
}

type Server interface {
	Serve(ctx context.Context) error
}

func NewServer(cfg config.APIConfig) Server {
	return &server{
		cfg: cfg,
	}
}

func (s *server) Serve(ctx context.Context) error {
	log := logging.LoggerFromContext(ctx)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	mux := http.NewServeMux()

	mux.Handle(grpchealth.NewHandler(NewHealthChecker()))

	srv := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	errCh := make(chan error)
	go func() { errCh <- srv.ListenAndServe() }()

	log.Infof("Server is listening on %q", addr)

	select {
	case <-ctx.Done():
		log.Info("Gracefully stopping server...")
		time.Sleep(s.cfg.GracefulShutdownTimeout)
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
