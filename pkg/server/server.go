package server

import (
	"bytes"
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

	"github.com/klauspost/compress/gzhttp"
	"github.com/rs/cors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/service/v1alpha1/svcv1alpha1connect"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	rollouts "github.com/akuity/kargo/pkg/api/stubs/rollouts"
	"github.com/akuity/kargo/pkg/event"
	httputil "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	libargocd "github.com/akuity/kargo/pkg/server/argocd"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/dex"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/option"
	"github.com/akuity/kargo/pkg/server/rbac"
	"github.com/akuity/kargo/pkg/server/validation"
)

var (
	_ svcv1alpha1connect.KargoServiceHandler = &server{}

	//go:embed all:ui
	ui embed.FS
)

type server struct {
	cfg            config.ServerConfig
	client         kubernetes.Client
	rolesDB        rbac.RolesDatabase
	sender         event.Sender
	argoCDURLStore libargocd.URLStore

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
	isFreightAvailableFn func(*kargoapi.Stage, *kargoapi.Freight) bool

	// Common Promotions:
	createPromotionFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	// Promote downstream:
	findDownstreamStagesFn func(
		context.Context,
		*kargoapi.Stage,
		kargoapi.FreightOrigin,
	) ([]kargoapi.Stage, error)

	// QueryFreight API:
	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
	getAvailableFreightForStageFn func(
		context.Context,
		*kargoapi.Stage,
	) ([]kargoapi.Freight, error)
	getFreightFromWarehousesFn func(
		ctx context.Context,
		project string,
		warehouses []string,
	) ([]kargoapi.Freight, error)
	getVerifiedFreightFn func(
		ctx context.Context,
		project string,
		upstreams []string,
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
	) (*rolloutsapi.AnalysisTemplate, error)

	getClusterAnalysisTemplateFn func(
		context.Context,
		client.Client,
		string,
	) (*rolloutsapi.ClusterAnalysisTemplate, error)

	getAnalysisRunFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rolloutsapi.AnalysisRun, error)

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
	rolesDB rbac.RolesDatabase,
	sender event.Sender,
	argoCDURLStore libargocd.URLStore,
) Server {
	s := &server{
		cfg:            cfg,
		client:         kubeClient,
		rolesDB:        rolesDB,
		sender:         sender,
		argoCDURLStore: argoCDURLStore,
	}

	s.validateProjectExistsFn = s.validateProjectExists
	s.externalValidateProjectFn = validation.ValidateProject
	s.getStageFn = api.GetStage
	s.getFreightByNameOrAliasFn = api.GetFreightByNameOrAlias
	s.isFreightAvailableFn = s.isFreightAvailable
	s.createPromotionFn = kubeClient.Create
	s.findDownstreamStagesFn = s.findDownstreamStages
	s.listFreightFn = kubeClient.List
	s.getAvailableFreightForStageFn = s.getAvailableFreightForStage
	s.getFreightFromWarehousesFn = s.getFreightFromWarehouses
	s.getVerifiedFreightFn = s.getVerifiedFreight
	s.patchFreightAliasFn = s.patchFreightAlias
	s.patchFreightStatusFn = s.patchFreightStatus
	s.authorizeFn = kubeClient.Authorize
	s.getAnalysisTemplateFn = rollouts.GetAnalysisTemplate
	s.getClusterAnalysisTemplateFn = rollouts.GetClusterAnalysisTemplate
	s.getAnalysisRunFn = rollouts.GetAnalysisRun

	return s
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	logger := logging.LoggerFromContext(ctx)
	mux := http.NewServeMux()

	opts, err := option.NewHandlerOption(ctx, s.cfg, s.client.InternalClient())
	if err != nil {
		return fmt.Errorf("error initializing handler options: %w", err)
	}
	mux.Handle("/healthz", newHealthHandler())
	path, svcHandler := svcv1alpha1connect.NewKargoServiceHandler(s, opts)
	mux.Handle(path, svcHandler)

	for p, h := range s.cfg.AdditionalHandlers {
		mux.Handle(p, h)
	}

	// Add Gin REST router
	ginRouter := s.setupRESTRouter(ctx)
	mux.Handle("/v1beta1/", ginRouter)

	var dashboardFS fs.FS
	if s.cfg.DashboardFS != nil {
		dashboardFS = s.cfg.DashboardFS
	} else {
		dashboardFS, err = fs.Sub(ui, "ui")
		if err != nil {
			return fmt.Errorf("error initializing UI file system: %w", err)
		}
	}
	mux.Handle("/", newDashboardRequestHandler(dashboardFS, s.cfg.BasePath))

	handler := wrapWithBasePath(mux, s.cfg.BasePath)

	// Dex's own configured issuer URL includes the API server's basePath, so
	// Dex routes its endpoints under that prefix on its own listener. Mount
	// the reverse proxy on an outer mux at the prefixed path so the basePath
	// is preserved end-to-end rather than getting stripped before forwarding.
	if s.cfg.DexProxyConfig != nil {
		dexProxy, err := dex.NewProxy(dex.ProxyConfigFromEnv())
		if err != nil {
			return fmt.Errorf("error initializing dex proxy: %w", err)
		}
		outer := http.NewServeMux()
		outer.Handle(s.cfg.BasePath+"/dex/", dexProxy)
		outer.Handle("/", handler)
		handler = outer
	}

	// Sometimes a permissive CORS policy is useful during local development.
	if s.cfg.PermissiveCORSPolicyEnabled {
		handler = cors.New(cors.Options{
			AllowCredentials: true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"DELETE", "GET", "POST", "PUT"},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
		}).Handler(handler)
	}

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Minute,
		Protocols:         protocols,
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

// wrapWithBasePath returns inner if basePath is empty, or otherwise an
// http.Handler that mounts inner under basePath via http.StripPrefix —
// inner's handlers continue to register at root-relative paths and the
// StripPrefix layer removes the basePath from each incoming URL before
// dispatching.
//
// The health check endpoint stays addressable at the root in parallel with
// the basePath-wrapped routes, so liveness / readiness probes that hit the
// Pod directly (never traversing the ingress that would otherwise prepend
// the basePath) keep working without basePath awareness.
func wrapWithBasePath(inner http.Handler, basePath string) http.Handler {
	if basePath == "" {
		return inner
	}
	stripped := http.StripPrefix(basePath, inner)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			inner.ServeHTTP(w, r)
			return
		}
		// Bare basePath (e.g. "/my-kargo") with no trailing slash: redirect
		// to "/my-kargo/" ourselves. Otherwise StripPrefix would leave the
		// inner mux with an empty path, and its canonical-path redirect
		// would point at "/" (which has no basePath context).
		if r.URL.Path == basePath {
			u := *r.URL
			u.Path = basePath + "/"
			http.Redirect(w, r, u.RequestURI(), http.StatusMovedPermanently)
			return
		}
		stripped.ServeHTTP(w, r)
	})
}

// indexHTMLBasePlaceholder is the literal string the bundled UI's index.html
// is expected to contain in the `<base href>` slot. The dashboard handler
// substitutes the runtime basePath into this placeholder on each serve,
// allowing a single bundled artifact to serve correctly under any prefix.
// When the placeholder is absent (e.g. an older UI build), the HTML is
// served unchanged.
const indexHTMLBasePlaceholder = "__BASE_HREF__"

// indexHTMLBasePathPlaceholder is the literal string in the bundled UI's
// index.html that the dashboard handler replaces with the runtime basePath,
// surfaced to the UI as `window.__KARGO_BASE_PATH__` so the React app can
// configure routing and absolute-URL construction against the deployed
// prefix.
const indexHTMLBasePathPlaceholder = "__BASE_PATH__"

func newDashboardRequestHandler(uiFS fs.FS, basePath string) http.HandlerFunc {
	const indexHTML = "index.html"

	// Pre-render the index.html body once, substituting the basePath
	// placeholders. The render result is small and held in memory for the
	// life of the process so we don't pay the read+replace cost on every
	// request that falls through to index.html.
	renderedIndex, indexLastModified := renderIndexHTML(uiFS, indexHTML, basePath)
	serveIndex := func(w http.ResponseWriter, req *http.Request) {
		httputil.SetNoCacheHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, req, indexHTML, indexLastModified, bytes.NewReader(renderedIndex))
	}

	handler := http.FileServer(http.FS(uiFS))
	withoutGzip := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		path := filepath.Clean(req.URL.Path)
		if path == "/" {
			serveIndex(w, req)
			return
		}

		f, err := uiFS.Open(strings.TrimPrefix(path, "/"))
		if f != nil {
			defer f.Close()
		}
		if os.IsNotExist(err) {
			// When the path doesn't match an embedded file, serve index.html
			serveIndex(w, req)
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
			serveIndex(w, req)
			return
		}

		// Path is a file
		httputil.SetCacheHeaders(w, 30*24*time.Hour, 7*24*time.Hour)
		handler.ServeHTTP(w, req)
	})

	withGz := gzhttp.GzipHandler(withoutGzip)
	return http.HandlerFunc(withGz.ServeHTTP)
}

// renderIndexHTML reads the dashboard's index.html out of uiFS, substitutes
// the basePath placeholders, and returns the rendered bytes plus a
// last-modified timestamp suitable for http.ServeContent. Returns an
// empty-bodied result and a zero timestamp if the file can't be read; the
// dashboard handler degrades to serving an empty document rather than
// returning errors at request time.
func renderIndexHTML(uiFS fs.FS, name, basePath string) ([]byte, time.Time) {
	raw, err := fs.ReadFile(uiFS, name)
	if err != nil {
		return nil, time.Time{}
	}
	baseHref := "/"
	basePathValue := ""
	if basePath != "" {
		baseHref = basePath + "/"
		basePathValue = basePath
	}
	rendered := bytes.ReplaceAll(raw, []byte(indexHTMLBasePlaceholder), []byte(baseHref))
	rendered = bytes.ReplaceAll(rendered, []byte(indexHTMLBasePathPlaceholder), []byte(basePathValue))
	// The embedded FS doesn't carry meaningful mtimes; pretend the file was
	// stamped at process start so ServeContent's If-Modified-Since logic is
	// well-defined.
	return rendered, time.Now()
}
