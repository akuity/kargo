package api

import (
	"context"
	"net"
	"net/http"
	goos "os"
	"time"

	"connectrpc.com/grpchealth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/dex"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/option"
	httputil "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/kubeclient/manifest"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}
)

type server struct {
	cfg    config.ServerConfig
	client kubernetes.Client

	parseKubernetesManifest manifest.ParseFunc
}

type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

func NewServer(
	cfg config.ServerConfig,
	client kubernetes.Client,
) (Server, error) {
	return &server{
		cfg:    cfg,
		client: client,

		parseKubernetesManifest: manifest.NewParser(client.Scheme()),
	}, nil
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	log := logging.LoggerFromContext(ctx)
	mux := http.NewServeMux()

	opts, err := option.NewHandlerOption(ctx, s.cfg)
	if err != nil {
		return errors.Wrap(err, "error initializing handler options")
	}
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

	log.WithFields(logrus.Fields{
		"tls": s.cfg.TLSConfig != nil,
	}).Infof("Server is listening on %q", l.Addr().String())

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
