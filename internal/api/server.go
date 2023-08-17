package api

import (
	"context"
	"net"
	"net/http"
	goos "os"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/dex"
	"github.com/akuity/kargo/internal/api/handler"
	"github.com/akuity/kargo/internal/api/option"
	httputil "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/version"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}
)

type server struct {
	cfg        config.ServerConfig
	kubeCli    client.Client
	dynamicCli dynamic.Interface
}

type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

func NewServer(
	cfg config.ServerConfig,
	kc client.Client,
	dynamicCli dynamic.Interface,
) (Server, error) {
	return &server{
		cfg:        cfg,
		kubeCli:    kc,
		dynamicCli: dynamicCli,
	}, nil
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	log := logging.LoggerFromContext(ctx)
	mux := http.NewServeMux()

	opts := option.NewHandlerOption(ctx, s.cfg)
	mux.Handle(grpchealth.NewHandler(NewHealthChecker(), opts))
	path, svcHandler := svcv1alpha1connect.NewKargoServiceHandler(s, opts)
	mux.Handle(path, svcHandler)
	mux.Handle("/", s.newDashboardRequestHandler())
	if s.cfg.DexProxyConfig != nil {
		dexProxyCfg := dex.ProxyConfigFromEnv()
		dexProxy, err := dex.NewProxy(dexProxyCfg)
		if err != nil {
			return errors.Wrap(err, "error initializing dex proxy")
		}
		mux.Handle("/dex/", dexProxy)
	}

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

func (s *server) newDashboardRequestHandler() http.HandlerFunc {
	fs := http.FileServer(http.Dir(s.cfg.UIDirectory))
	return func(w http.ResponseWriter, req *http.Request) {
		path := s.cfg.UIDirectory + req.URL.Path
		info, err := goos.Stat(path)
		if goos.IsNotExist(err) || info.IsDir() {
			if w != nil {
				httputil.SetNoCacheHeaders(w)
				http.ServeFile(w, req, s.cfg.UIDirectory+"/index.html")
			}
		} else {
			fs.ServeHTTP(w, req)
		}
	}
}

func (s *server) GetVersionInfo(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetVersionInfoRequest],
) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
	return handler.GetVersionInfoV1Alpha1(version.GetVersion())(ctx, req)
}

func (s *server) GetPublicConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetPublicConfigRequest],
) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error) {
	return handler.GetPublicConfigV1Alpha1(s.cfg)(ctx, req)
}

func (s *server) AdminLogin(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.AdminLoginRequest],
) (*connect.Response[svcv1alpha1.AdminLoginResponse], error) {
	return handler.AdminLoginV1Alpha1(s.cfg.AdminConfig)(ctx, req)
}

func (s *server) CreateStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateStageRequest],
) (*connect.Response[svcv1alpha1.CreateStageResponse], error) {
	return handler.CreateStageV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) ListStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesResponse], error) {
	return handler.ListStagesV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) GetStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetStageRequest],
) (*connect.Response[svcv1alpha1.GetStageResponse], error) {
	return handler.GetStageV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) WatchStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchStagesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchStagesResponse],
) error {
	return handler.WatchStageV1Alpha1(s.kubeCli, s.dynamicCli)(ctx, req, stream)
}

func (s *server) UpdateStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateStageRequest],
) (*connect.Response[svcv1alpha1.UpdateStageResponse], error) {
	return handler.UpdateStageV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) DeleteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteStageRequest],
) (*connect.Response[svcv1alpha1.DeleteStageResponse], error) {
	return handler.DeleteStageV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) PromoteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteStageRequest],
) (*connect.Response[svcv1alpha1.PromoteStageResponse], error) {
	return handler.PromoteStageV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) SetAutoPromotionForStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.SetAutoPromotionForStageRequest],
) (*connect.Response[svcv1alpha1.SetAutoPromotionForStageResponse], error) {
	return handler.SetAutoPromotionForStageV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) CreatePromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreatePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.CreatePromotionPolicyResponse], error) {
	return handler.CreatePromotionPolicyV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) ListPromotionPolicies(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListPromotionPoliciesRequest],
) (*connect.Response[svcv1alpha1.ListPromotionPoliciesResponse], error) {
	return handler.ListPromotionPoliciesV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) GetPromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetPromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.GetPromotionPolicyResponse], error) {
	return handler.GetPromotionPolicyV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) UpdatePromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdatePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.UpdatePromotionPolicyResponse], error) {
	return handler.UpdatePromotionPolicyV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) DeletePromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeletePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.DeletePromotionPolicyResponse], error) {
	return handler.DeletePromotionPolicyV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) CreateProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateProjectRequest],
) (*connect.Response[svcv1alpha1.CreateProjectResponse], error) {
	return handler.CreateProjectV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) ListProjects(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	return handler.ListProjectsV1Alpha1(s.kubeCli)(ctx, req)
}

func (s *server) DeleteProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteProjectRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectResponse], error) {
	return handler.DeleteProjectV1Alpha1(s.kubeCli)(ctx, req)
}
