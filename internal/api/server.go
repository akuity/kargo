package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/handler"
	"github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/logging"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}
)

type server struct {
	cfg config.APIConfig
	kc  client.Client
}

type Server interface {
	Serve(ctx context.Context, l net.Listener, localMode bool) error
}

func NewServer(kc client.Client, cfg config.APIConfig) (Server, error) {
	return &server{
		cfg: cfg,
		kc:  kc,
	}, nil
}

func (s *server) Serve(
	ctx context.Context,
	l net.Listener,
	localMode bool,
) error {
	log := logging.LoggerFromContext(ctx)
	mux := http.NewServeMux()

	opts := option.NewHandlerOption(ctx, localMode)
	mux.Handle(grpchealth.NewHandler(NewHealthChecker(), opts))
	path, svcHandler := svcv1alpha1connect.NewKargoServiceHandler(s, opts)
	mux.Handle(path, svcHandler)

	srv := &http.Server{
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: time.Minute,
	}

	errCh := make(chan error)
	go func() { errCh <- srv.Serve(l) }()

	log.Infof("Server is listening on %q", l.Addr().String())

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

func (s *server) CreateEnvironment(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateEnvironmentRequest],
) (*connect.Response[svcv1alpha1.CreateEnvironmentResponse], error) {
	return handler.CreateEnvironmentV1Alpha1(s.kc)(ctx, req)
}

func (s *server) ListEnvironments(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListEnvironmentsRequest],
) (*connect.Response[svcv1alpha1.ListEnvironmentsResponse], error) {
	return handler.ListEnvironmentsV1Alpha1(s.kc)(ctx, req)
}

func (s *server) GetEnvironment(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetEnvironmentRequest],
) (*connect.Response[svcv1alpha1.GetEnvironmentResponse], error) {
	return handler.GetEnvironmentV1Alpha1(s.kc)(ctx, req)
}

func (s *server) DeleteEnvironment(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteEnvironmentRequest],
) (*connect.Response[svcv1alpha1.DeleteEnvironmentResponse], error) {
	return handler.DeleteEnvironmentV1Alpha1(s.kc)(ctx, req)
}

func (s *server) PromoteEnvironment(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteEnvironmentRequest],
) (*connect.Response[svcv1alpha1.PromoteEnvironmentResponse], error) {
	return handler.PromoteEnvironmentV1Alpha1(s.kc)(ctx, req)
}
