package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/logging"
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
	log.Infof("Server is listening on %q", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "listen %s", addr)
	}

	srv := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, newGRPCHealthV1Server())

	errCh := make(chan error)
	go func() { errCh <- srv.Serve(l) }()

	select {
	case <-ctx.Done():
		log.Info("Gracefully stopping server...")
		time.Sleep(s.cfg.GracefulShutdownTimeout)
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}
