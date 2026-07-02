package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/argocd"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/rbac"
)

func TestNewServer(t *testing.T) {
	testServerConfig := config.ServerConfig{}
	testClient, err := kubernetes.NewClient(
		t.Context(),
		&rest.Config{},
		kubernetes.ClientOptions{
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
				string,
			) (client.WithWatch, error) {
				return fake.NewClientBuilder().Build(), nil
			},
		},
	)
	require.NoError(t, err)
	testSender := k8sevent.NewEventSender(fakeevent.NewEventRecorder(0))
	testURLStore := argocd.NewURLStore()

	s, ok := NewServer(
		testServerConfig,
		testClient,
		rbac.NewKubernetesRolesDatabase(
			testClient,
			testClient,
			rbac.RolesDatabaseConfigFromEnv(),
		),
		testSender,
		testURLStore,
	).(*server)

	require.True(t, ok)
	require.NotNil(t, s)
	require.Same(t, testClient, s.client)
	require.NotNil(t, testClient, s.rolesDB)
	require.Same(t, testSender, s.sender)
	require.Same(t, testURLStore, s.argoCDURLStore)
	require.Equal(t, testServerConfig, s.cfg)
	require.NotNil(t, s.validateProjectExistsFn)
	require.NotNil(t, s.externalValidateProjectFn)
	require.NotNil(t, s.getStageFn)
	require.NotNil(t, s.getFreightByNameOrAliasFn)
	require.NotNil(t, s.isFreightAvailableFn)
	require.NotNil(t, s.createPromotionFn)
	require.NotNil(t, s.findDownstreamStagesFn)
	require.NotNil(t, s.listFreightFn)
	require.NotNil(t, s.getAvailableFreightForStageFn)
	require.NotNil(t, s.getFreightFromWarehousesFn)
	require.NotNil(t, s.getVerifiedFreightFn)
	require.NotNil(t, s.patchFreightAliasFn)
	require.NotNil(t, s.patchFreightStatusFn)
	require.NotNil(t, s.authorizeFn)
	require.NotNil(t, s.getAnalysisRunFn)
}

func TestWrapWithBasePath(t *testing.T) {
	// Inner mux registers handlers at root paths; the wrap moves them under
	// basePath with the carve-out for gRPC health.
	innerMux := http.NewServeMux()
	innerMux.HandleFunc("/grpc.health.v1.Health/Check", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("health-ok"))
	})
	innerMux.HandleFunc("/v1beta1/projects", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("projects-at-" + r.URL.Path))
	})

	testCases := []struct {
		name         string
		basePath     string
		requestPath  string
		wantStatus   int
		wantBodyHas  string
		wantLocation string
	}{
		{
			name:        "no basePath: root request reaches inner handler",
			basePath:    "",
			requestPath: "/v1beta1/projects",
			wantStatus:  http.StatusOK,
			wantBodyHas: "projects-at-/v1beta1/projects",
		},
		{
			name:        "no basePath: health at root reaches handler",
			basePath:    "",
			requestPath: "/grpc.health.v1.Health/Check",
			wantStatus:  http.StatusOK,
			wantBodyHas: "health-ok",
		},
		{
			name:        "basePath set: prefixed request strips prefix and reaches inner handler",
			basePath:    "/kargo",
			requestPath: "/kargo/v1beta1/projects",
			wantStatus:  http.StatusOK,
			wantBodyHas: "projects-at-/v1beta1/projects",
		},
		{
			name:        "basePath set: unprefixed v1beta1 returns 404",
			basePath:    "/kargo",
			requestPath: "/v1beta1/projects",
			wantStatus:  http.StatusNotFound,
		},
		{
			name:        "basePath set: health at root still reaches handler (probes don't traverse ingress)",
			basePath:    "/kargo",
			requestPath: "/grpc.health.v1.Health/Check",
			wantStatus:  http.StatusOK,
			wantBodyHas: "health-ok",
		},
		{
			name:        "basePath set: health under basePath also works",
			basePath:    "/kargo",
			requestPath: "/kargo/grpc.health.v1.Health/Check",
			wantStatus:  http.StatusOK,
			wantBodyHas: "health-ok",
		},
		{
			name:        "multi-segment basePath: prefixed request reaches inner handler",
			basePath:    "/teams/kargo",
			requestPath: "/teams/kargo/v1beta1/projects",
			wantStatus:  http.StatusOK,
			wantBodyHas: "projects-at-/v1beta1/projects",
		},
		{
			name:         "bare basePath without trailing slash redirects to slashed form",
			basePath:     "/kargo",
			requestPath:  "/kargo",
			wantStatus:   http.StatusMovedPermanently,
			wantLocation: "/kargo/",
		},
		{
			name:         "bare multi-segment basePath also redirects",
			basePath:     "/teams/kargo",
			requestPath:  "/teams/kargo",
			wantStatus:   http.StatusMovedPermanently,
			wantLocation: "/teams/kargo/",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := wrapWithBasePath(innerMux, tc.basePath)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.requestPath, nil)
			handler.ServeHTTP(rec, req)
			require.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantBodyHas != "" {
				require.Contains(t, rec.Body.String(), tc.wantBodyHas)
			}
			if tc.wantLocation != "" {
				require.Equal(t, tc.wantLocation, rec.Header().Get("Location"))
			}
		})
	}
}

func TestRenderIndexHTML(t *testing.T) {
	const indexBody = `<!DOCTYPE html>
<html>
<head>
  <base href="__BASE_HREF__">
  <script>window.__KARGO_BASE_PATH__ = "__BASE_PATH__";</script>
  <link rel="icon" href="favicon.png">
</head>
<body><div id="root"></div></body>
</html>`

	testCases := []struct {
		name          string
		basePath      string
		indexContents string
		fileName      string
		wantBaseHref  string
		wantBasePath  string
		wantBodyEmpty bool
	}{
		{
			name:          "empty basePath substitutes root href and empty global",
			basePath:      "",
			indexContents: indexBody,
			fileName:      "index.html",
			wantBaseHref:  `<base href="/">`,
			wantBasePath:  `window.__KARGO_BASE_PATH__ = "";`,
		},
		{
			name:          "non-empty basePath substitutes prefix",
			basePath:      "/kargo",
			indexContents: indexBody,
			fileName:      "index.html",
			wantBaseHref:  `<base href="/kargo/">`,
			wantBasePath:  `window.__KARGO_BASE_PATH__ = "/kargo";`,
		},
		{
			name:          "multi-segment basePath",
			basePath:      "/teams/kargo",
			indexContents: indexBody,
			fileName:      "index.html",
			wantBaseHref:  `<base href="/teams/kargo/">`,
			wantBasePath:  `window.__KARGO_BASE_PATH__ = "/teams/kargo";`,
		},
		{
			name:          "html without placeholders is served unchanged",
			basePath:      "/kargo",
			indexContents: `<html><body>no placeholders</body></html>`,
			fileName:      "index.html",
			wantBaseHref:  "",
			wantBasePath:  "",
		},
		{
			name:          "missing file returns empty body",
			basePath:      "/kargo",
			indexContents: indexBody,
			fileName:      "does-not-exist.html",
			wantBodyEmpty: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"index.html": &fstest.MapFile{Data: []byte(tc.indexContents)},
			}
			body, ts := renderIndexHTML(fsys, tc.fileName, tc.basePath)
			if tc.wantBodyEmpty {
				require.Empty(t, body)
				require.True(t, ts.IsZero())
				return
			}
			require.False(t, ts.IsZero())
			rendered := string(body)
			require.NotContains(t, rendered, indexHTMLBasePlaceholder,
				"placeholder __BASE_HREF__ should have been substituted")
			require.NotContains(t, rendered, indexHTMLBasePathPlaceholder,
				"placeholder __BASE_PATH__ should have been substituted")
			if tc.wantBaseHref != "" {
				require.Contains(t, rendered, tc.wantBaseHref)
			}
			if tc.wantBasePath != "" {
				require.Contains(t, rendered, tc.wantBasePath)
			}
		})
	}
}
