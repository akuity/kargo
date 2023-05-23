package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/handler"
	"github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/kubeclient"
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
	Serve(ctx context.Context) error
}

func NewServer(cfg config.APIConfig) (Server, error) {
	var rc *rest.Config
	var err error
	if cfg.LocalMode {
		rc, err = kubeclient.NewClientConfig().ClientConfig()
	} else {
		rc, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, errors.Wrap(err, "load kubeconfig")
	}

	scheme := runtime.NewScheme()
	if err = corev1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add core api to scheme")
	}
	if err = kubev1alpha1.AddToScheme(scheme); err != nil {
		return nil, errors.Wrap(err, "add kargo api to scheme")
	}

	kc, err := client.New(rc, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}
	return &server{
		cfg: cfg,
		kc:  kc,
	}, nil
}

func (s *server) Serve(ctx context.Context) error {
	log := logging.LoggerFromContext(ctx)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	mux := http.NewServeMux()

	opts := option.NewHandlerOption(s.cfg, log)
	mux.Handle(grpchealth.NewHandler(NewHealthChecker(), opts))
	path, svcHandler := svcv1alpha1connect.NewKargoServiceHandler(s, opts)
	mux.Handle(path, svcHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: time.Minute,
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
