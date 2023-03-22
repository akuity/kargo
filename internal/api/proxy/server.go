package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/logging"
)

type server struct {
	cfg config.APIProxyConfig
}

type Server interface {
	Serve(ctx context.Context) error
}

func NewServer(cfg config.APIProxyConfig) Server {
	return &server{
		cfg: cfg,
	}
}

func (s *server) Serve(ctx context.Context) error {
	log := logging.LoggerFromContext(ctx)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	log.Infof("Proxy is listening on %q", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "listen %s", addr)
	}

	cc, err := grpc.Dial(s.cfg.APIEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return errors.Wrap(err, "dial API endpoint")
	}

	mux := runtime.NewServeMux(
		runtime.WithHealthzEndpoint(grpc_health_v1.NewHealthClient(cc)),
	)
	// TODO: Register services
	srv := &http.Server{
		Handler: mux,
	}
	errCh := make(chan error)
	go func() { errCh <- srv.Serve(l) }()

	select {
	case <-ctx.Done():
		log.Info("Gracefully stopping server...")
		time.Sleep(s.cfg.GracefulShutdownTimeout)
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}
