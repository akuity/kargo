package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/handler"
	"github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/logging"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}
)

type ServerConfig struct {
	GracefulShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT" default:"30s"`
}

func ServerConfigFromEnv() ServerConfig {
	cfg := ServerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type server struct {
	cfg ServerConfig
	kc  client.Client
}

type Server interface {
	Serve(ctx context.Context, l net.Listener, localMode bool) error
}

func NewServer(kc client.Client, cfg ServerConfig) (Server, error) {
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

func (s *server) CreateStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateStageRequest],
) (*connect.Response[svcv1alpha1.CreateStageResponse], error) {
	return handler.CreateStageV1Alpha1(s.kc)(ctx, req)
}

func (s *server) ListStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesResponse], error) {
	return handler.ListStagesV1Alpha1(s.kc)(ctx, req)
}

func (s *server) GetStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetStageRequest],
) (*connect.Response[svcv1alpha1.GetStageResponse], error) {
	return handler.GetStageV1Alpha1(s.kc)(ctx, req)
}

func (s *server) UpdateStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateStageRequest],
) (*connect.Response[svcv1alpha1.UpdateStageResponse], error) {
	return handler.UpdateStageV1Alpha1(s.kc)(ctx, req)
}

func (s *server) DeleteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteStageRequest],
) (*connect.Response[svcv1alpha1.DeleteStageResponse], error) {
	return handler.DeleteStageV1Alpha1(s.kc)(ctx, req)
}

func (s *server) PromoteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteStageRequest],
) (*connect.Response[svcv1alpha1.PromoteStageResponse], error) {
	return handler.PromoteStageV1Alpha1(s.kc)(ctx, req)
}

func (s *server) CreateProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateProjectRequest],
) (*connect.Response[svcv1alpha1.CreateProjectResponse], error) {
	return handler.CreateProjectV1Alpha1(s.kc)(ctx, req)
}

func (s *server) ListProjects(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	return handler.ListProjectsV1Alpha1(s.kc)(ctx, req)
}

func (s *server) DeleteProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteProjectRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectResponse], error) {
	return handler.DeleteProjectV1Alpha1(s.kc)(ctx, req)
}
