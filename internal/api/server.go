package api

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/grpchealth"
	"github.com/klauspost/compress/gzhttp"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/dex"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/api/rbac"
	"github.com/akuity/kargo/internal/api/validation"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	httputil "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}

	//go:embed all:ui
	ui embed.FS
)

type server struct {
	cfg            config.ServerConfig
	client         kubernetes.Client
	internalClient client.Client
	rolesDB        rbac.RolesDatabase
	recorder       record.EventRecorder

	// The following behaviors are overridable for testing purposes:

	// Common validations:
	validateProjectExistsFn func(
		ctx context.Context,
		project string,
	) error

	externalValidateProjectFn func(
		ctx context.Context,
		client client.Client,
		project string,
	) error

	// Common lookups:
	getStageFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Stage, error)
	getFreightByNameOrAliasFn func(
		ctx context.Context,
		c client.Client,
		project string,
		name string,
		alias string,
	) (*kargoapi.Freight, error)
	isFreightAvailableFn func(
		freight *kargoapi.Freight,
		stage string,
		upstreamStages []string,
	) bool

	// Common Promotions:
	createPromotionFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	// Promote subscribers:
	findStageSubscribersFn func(ctx context.Context, stage *kargoapi.Stage) ([]kargoapi.Stage, error)

	// QueryFreight API:
	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
	getAvailableFreightForStageFn func(
		ctx context.Context,
		project string,
		stage string,
		subs kargoapi.Subscriptions,
	) ([]kargoapi.Freight, error)
	getFreightFromWarehouseFn func(
		ctx context.Context,
		project string,
		warehouse string,
	) ([]kargoapi.Freight, error)
	getVerifiedFreightFn func(
		ctx context.Context,
		project string,
		stageSubs []kargoapi.StageSubscription,
	) ([]kargoapi.Freight, error)

	// Freight aliasing:
	patchFreightAliasFn func(
		ctx context.Context,
		freight *kargoapi.Freight,
		alias string,
	) error

	// Freight approval:
	patchFreightStatusFn func(
		ctx context.Context,
		freight *kargoapi.Freight,
		newStatus kargoapi.FreightStatus,
	) error

	// Rollouts integration:
	getAnalysisTemplateFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rollouts.AnalysisTemplate, error)

	getAnalysisRunFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rollouts.AnalysisRun, error)

	// Special authorizations:
	authorizeFn func(
		ctx context.Context,
		verb string,
		gvr schema.GroupVersionResource,
		subresource string,
		key client.ObjectKey,
	) error
}

type Server interface {
	Serve(ctx context.Context, l net.Listener) error
}

func NewServer(
	cfg config.ServerConfig,
	kubeClient kubernetes.Client,
	internalClient client.Client,
	rolesDB rbac.RolesDatabase,
	recorder record.EventRecorder,
) Server {
	s := &server{
		cfg:            cfg,
		client:         kubeClient,
		internalClient: internalClient,
		rolesDB:        rolesDB,
		recorder:       recorder,
	}

	s.validateProjectExistsFn = s.validateProjectExists
	s.externalValidateProjectFn = validation.ValidateProject
	s.getStageFn = kargoapi.GetStage
	s.getFreightByNameOrAliasFn = kargoapi.GetFreightByNameOrAlias
	s.isFreightAvailableFn = kargoapi.IsFreightAvailable
	s.createPromotionFn = kubeClient.Create
	s.findStageSubscribersFn = s.findStageSubscribers
	s.listFreightFn = kubeClient.List
	s.getAvailableFreightForStageFn = s.getAvailableFreightForStage
	s.getFreightFromWarehouseFn = s.getFreightFromWarehouse
	s.getVerifiedFreightFn = s.getVerifiedFreight
	s.patchFreightAliasFn = s.patchFreightAlias
	s.patchFreightStatusFn = s.patchFreightStatus
	s.authorizeFn = kubeClient.Authorize
	s.getAnalysisTemplateFn = rollouts.GetAnalysisTemplate
	s.getAnalysisRunFn = rollouts.GetAnalysisRun

	return s
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	logger := logging.LoggerFromContext(ctx)
	mux := http.NewServeMux()

	opts, err := option.NewHandlerOption(ctx, s.cfg, s.internalClient)
	if err != nil {
		return fmt.Errorf("error initializing handler options: %w", err)
	}
	mux.Handle(grpchealth.NewHandler(NewHealthChecker(), opts))
	path, svcHandler := svcv1alpha1connect.NewKargoServiceHandler(s, opts)
	mux.Handle(path, svcHandler)
	dashboardHandler, err := newDashboardRequestHandler()
	if err != nil {
		return fmt.Errorf("error initializing dashboard handler: %w", err)
	}
	mux.Handle("/", dashboardHandler)
	if s.cfg.DexProxyConfig != nil {
		dexProxyCfg := dex.ProxyConfigFromEnv()
		dexProxy, err := dex.NewProxy(dexProxyCfg)
		if err != nil {
			return fmt.Errorf("error initializing dex proxy: %w", err)
		}
		mux.Handle("/dex/", dexProxy)
	}

	handler := h2c.NewHandler(mux, &http2.Server{})

	// Sometimes a permissive CORS policy is useful during local development.
	if s.cfg.PermissiveCORSPolicyEnabled {
		handler = cors.New(cors.Options{
			AllowCredentials: true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"DELETE", "GET", "POST", "PUT"},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
		}).Handler(handler)
	}

	srv := &http.Server{
		Handler:           handler,
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

func newDashboardRequestHandler() (http.HandlerFunc, error) {
	const indexHTML = "index.html"

	uiFS := fs.FS(ui)
	uiFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return nil, fmt.Errorf("error initializing UI file system: %w", err)
	}

	handler := http.FileServer(http.FS(uiFS))
	withoutGzip := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		path := filepath.Clean(req.URL.Path)
		if path == "/" {
			httputil.SetNoCacheHeaders(w)
			http.ServeFileFS(w, req, uiFS, indexHTML)
			return
		}

		f, err := uiFS.Open(strings.TrimPrefix(path, "/"))
		if f != nil {
			defer f.Close()
		}
		if os.IsNotExist(err) {
			// When the path doesn't match an embedded file, serve index.html
			httputil.SetNoCacheHeaders(w)
			http.ServeFileFS(w, req, uiFS, indexHTML)
			return
		}
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// If we get to here, the path exists in the embedded file system

		info, err := f.Stat()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if info.IsDir() {
			// Serve index.html to prevent enumerating files in the directory
			httputil.SetNoCacheHeaders(w)
			http.ServeFileFS(w, req, uiFS, indexHTML)
			return
		}

		// Path is a file
		handler.ServeHTTP(w, req)
	})

	withGz := gzhttp.GzipHandler(withoutGzip)
	return http.HandlerFunc(withGz.ServeHTTP), nil
}
