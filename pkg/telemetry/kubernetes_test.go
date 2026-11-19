package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/rest"

	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

func TestWrapRESTConfig_tracingOff(t *testing.T) {
	// Not parallel: reads global state other tests change.
	cfg := &rest.Config{}
	WrapRESTConfig(cfg)
	require.Nil(t, cfg.WrapTransport)
	WrapRESTConfig(nil)
}

func TestWrapRESTConfig_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	// k8s is off by default.
	for _, categories := range []string{"default", "database", "none"} {
		t.Run(categories, func(t *testing.T) {
			ended := telemetrytest.RecordSpans(t)
			telemetrytest.SetCategories(t, categories)
			var traceparent string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				traceparent = r.Header.Get("traceparent")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			cfg := &rest.Config{Host: srv.URL}
			WrapRESTConfig(cfg)
			require.Nil(t, cfg.WrapTransport)
			httpClient, err := rest.HTTPClientFor(cfg)
			require.NoError(t, err)
			ctx, parent := otel.Tracer("test").Start(context.Background(), "parent")
			defer parent.End()
			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				srv.URL+"/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/stages/dev",
				nil,
			)
			require.NoError(t, err)
			res, err := httpClient.Do(req)
			require.NoError(t, err)
			_ = res.Body.Close()
			require.Empty(t, traceparent)
			require.Empty(t, ended())
		})
	}
}

func TestWrapRESTConfig(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	var mu sync.Mutex
	traceparents := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		traceparents[r.URL.Path] = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &rest.Config{Host: srv.URL}
	WrapRESTConfig(cfg)
	httpClient, err := rest.HTTPClientFor(cfg)
	require.NoError(t, err)

	const (
		statusPath = "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc/status"
		leasePath  = "/apis/coordination.k8s.io/v1/namespaces/kargo/leases/kargo-controller"
		eventPath  = "/api/v1/namespaces/smoke/events"
		watchPath  = "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights?watch=true"
	)
	ctx := ContextWithAttributes(context.Background(), StageKey.String("dev"))
	for _, r := range []struct{ method, path string }{
		{http.MethodPatch, statusPath},
		{http.MethodPatch, leasePath},
		{http.MethodPost, eventPath},
		{http.MethodGet, watchPath},
	} {
		req, err := http.NewRequestWithContext(ctx, r.method, srv.URL+r.path, nil)
		require.NoError(t, err)
		res, err := httpClient.Do(req)
		require.NoError(t, err)
		_ = res.Body.Close()
	}

	// Leases, events and watches are not traced; a traced request carries its
	// span's trace context.
	mu.Lock()
	require.NotEmpty(t, traceparents[statusPath])
	require.Empty(t, traceparents[leasePath])
	require.Empty(t, traceparents[eventPath])
	mu.Unlock()
	spans := ended()
	require.Len(t, spans, 1)
	require.Equal(t, "k8s patch freights/status", spans[0].Name())
	require.Equal(t, trace.SpanKindClient, spans[0].SpanKind())
	require.Subset(
		t,
		spans[0].Attributes(),
		[]any{
			StageKey.String("dev"),
			ProjectKey.String("smoke"),
			FreightKey.String("abc"),
			VerbKey.String("patch"),
			SubresourceKey.String("status"),
		},
	)
}

func TestParseKubeRequest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		method   string
		url      string
		expected kubeRequest
	}{
		{
			method: http.MethodPatch,
			url:    "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc/status",
			expected: kubeRequest{
				verb:        "patch",
				group:       "kargo.akuity.io",
				namespace:   "smoke",
				resource:    "freights",
				name:        "abc",
				subresource: "status",
			},
		},
		{
			method: http.MethodGet,
			url:    "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/stages?watch=true",
			expected: kubeRequest{
				verb:      "watch",
				group:     "kargo.akuity.io",
				namespace: "smoke",
				resource:  "stages",
			},
		},
		{
			method:   http.MethodGet,
			url:      "/api/v1/namespaces/smoke",
			expected: kubeRequest{verb: "get", resource: "namespaces", name: "smoke"},
		},
		{
			method:   http.MethodPost,
			url:      "/api/v1/namespaces/smoke/events",
			expected: kubeRequest{verb: "create", namespace: "smoke", resource: "events"},
		},
		{
			method:   http.MethodGet,
			url:      "/version",
			expected: kubeRequest{verb: "get", resource: "other"},
		},
		{
			method:   http.MethodPost,
			url:      "/api/v1",
			expected: kubeRequest{verb: "post", resource: "other"},
		},
		{
			method:   http.MethodGet,
			url:      "/api/v1/namespaces/smoke/pods?labelSelector=a%3Db&watch=1",
			expected: kubeRequest{verb: "watch", namespace: "smoke", resource: "pods"},
		},
		{
			method: http.MethodGet,
			url:    "/api/v1/namespaces/smoke/pods/p/log/x",
			expected: kubeRequest{
				verb:        "get",
				namespace:   "smoke",
				resource:    "pods",
				name:        "p",
				subresource: "log",
			},
		},
		{
			method: http.MethodGet,
			url:    "/api/v1/namespaces/smoke/services/svc:80/proxy/a/b?c=d",
			expected: kubeRequest{
				verb:        "get",
				namespace:   "smoke",
				resource:    "services",
				name:        "svc:80",
				subresource: "proxy",
			},
		},
		{
			method:   http.MethodGet,
			url:      "/api/v1/watch/namespaces/smoke/pods",
			expected: kubeRequest{verb: "watch", namespace: "smoke", resource: "pods"},
		},
		{
			method:   http.MethodGet,
			url:      "/apis/kargo.akuity.io/v1alpha1/watch/stages",
			expected: kubeRequest{verb: "watch", group: "kargo.akuity.io", resource: "stages"},
		},
		{
			method: http.MethodPut,
			url:    "/api/v1/namespaces/smoke/status",
			expected: kubeRequest{
				verb:        "update",
				resource:    "namespaces",
				name:        "smoke",
				subresource: "status",
			},
		},
		{
			method: http.MethodPut,
			url:    "/api/v1/namespaces/smoke/finalize",
			expected: kubeRequest{
				verb:        "update",
				resource:    "namespaces",
				name:        "smoke",
				subresource: "finalize",
			},
		},
		{
			method: http.MethodDelete,
			url:    "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights",
			expected: kubeRequest{
				verb:      "deletecollection",
				group:     "kargo.akuity.io",
				namespace: "smoke",
				resource:  "freights",
			},
		},
		{
			method: http.MethodPut,
			url:    "/apis/kargo.akuity.io/v1alpha1/projects/smoke/",
			expected: kubeRequest{
				verb:     "update",
				group:    "kargo.akuity.io",
				resource: "projects",
				name:     "smoke",
			},
		},
		{
			method:   http.MethodPut,
			url:      "/version",
			expected: kubeRequest{verb: "put", resource: "other"},
		},
		{
			method:   http.MethodPatch,
			url:      "/version",
			expected: kubeRequest{verb: "patch", resource: "other"},
		},
		{
			method:   http.MethodDelete,
			url:      "/version",
			expected: kubeRequest{verb: "delete", resource: "other"},
		},
		{
			method:   "OPTIONS",
			url:      "/version",
			expected: kubeRequest{verb: "options", resource: "other"},
		},
		{
			method:   "OPTIONS",
			url:      "/api/v1/namespaces/smoke/pods",
			expected: kubeRequest{verb: "options", namespace: "smoke", resource: "pods"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.method+" "+testCase.url, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(testCase.method, testCase.url, nil)
			require.Equal(t, testCase.expected, parseKubeRequest(r, ""))
		})
	}
}

func TestParseKubeRequest_prefix(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(
		http.MethodGet,
		"/k8s/clusters/c1/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc",
		nil,
	)
	require.Equal(
		t,
		kubeRequest{
			verb:      "get",
			group:     "kargo.akuity.io",
			namespace: "smoke",
			resource:  "freights",
			name:      "abc",
		},
		parseKubeRequest(r, "k8s/clusters/c1/"),
	)
	require.Equal(t, kubeRequest{verb: "get", resource: "other"}, parseKubeRequest(r, ""))
}

func TestWrapRESTConfig_hostPath(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &rest.Config{Host: srv.URL + "/k8s/clusters/c1/"}
	WrapRESTConfig(cfg)
	httpClient, err := rest.HTTPClientFor(cfg)
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		srv.URL+"/k8s/clusters/c1/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/stages/dev",
		nil,
	)
	require.NoError(t, err)
	res, err := httpClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	span := telemetrytest.EndedSpan(t, ended(), "k8s get stages")
	require.Subset(t, span.Attributes(), []any{StageKey.String("dev")})
}

func TestParseKubeRequest_noAllocs(t *testing.T) {
	// Not parallel: AllocsPerRun panics in parallel tests.
	// It runs twice per traced request.
	r := httptest.NewRequest(
		http.MethodGet,
		"/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc/status?watch=false",
		nil,
	)
	require.Zero(t, testing.AllocsPerRun(100, func() { _ = parseKubeRequest(r, "") }))
}

func TestKubeRequest_attributes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		url      string
		expected []any
		absent   []any
	}{
		{
			name: "Kargo object",
			url:  "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/promotions/dev.01",
			expected: []any{
				VerbKey.String("get"),
				ResourceKey.String("promotions"),
				NameKey.String("dev.01"),
				ProjectKey.String("smoke"),
				PromotionKey.String("dev.01"),
			},
		},
		{
			name: "Freight",
			url:  "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc",
			expected: []any{
				ProjectKey.String("smoke"),
				FreightKey.String("abc"),
			},
		},
		{
			name: "Warehouse",
			url:  "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/warehouses/wh",
			expected: []any{
				ProjectKey.String("smoke"),
				WarehouseKey.String("wh"),
			},
		},
		{
			name: "Stage",
			url:  "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/stages/dev",
			expected: []any{
				ProjectKey.String("smoke"),
				StageKey.String("dev"),
			},
		},
		{
			name: "PromotionRequest",
			url:  "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/promotionrequests/req",
			expected: []any{
				ProjectKey.String("smoke"),
				PromotionRequestKey.String("req"),
			},
		},
		{
			name:     "Project",
			url:      "/apis/kargo.akuity.io/v1alpha1/projects/smoke",
			expected: []any{ProjectKey.String("smoke")},
		},
		{
			name: "Kargo list",
			url:  "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/stages",
			expected: []any{
				VerbKey.String("list"),
				ProjectKey.String("smoke"),
			},
		},
		{
			name: "other object",
			url:  "/api/v1/namespaces/kargo/configmaps/cfg",
			expected: []any{
				NameKey.String("cfg"),
				NamespaceKey.String("kargo"),
			},
			absent: []any{ProjectKey.String("kargo")},
		},
		{
			name: "Secret",
			url:  "/api/v1/namespaces/smoke/secrets/repo-creds",
			expected: []any{
				ResourceKey.String("secrets"),
				NamespaceKey.String("smoke"),
			},
			absent: []any{NameKey.String("repo-creds")},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, testCase.url, nil)
			attrs := parseKubeRequest(r, "").attributes()
			require.Subset(t, attrs, testCase.expected)
			for _, a := range testCase.absent {
				require.NotContains(t, attrs, a)
			}
		})
	}
}

func TestKubeAttributes_urlFull(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(
		http.MethodGet,
		"https://kube.example.com/api/v1/namespaces/smoke/secrets/repo-creds?resourceVersion=1",
		nil,
	)
	require.Contains(
		t,
		kubeAttributes(r, ""),
		semconv.URLFull("https://kube.example.com/api/v1/namespaces/smoke/secrets/{redacted}"),
	)
}

func TestWrapRESTConfig_status(t *testing.T) {
	const path = "/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc"
	testCases := []struct {
		name   string
		status int
		assert func(*testing.T, sdktrace.ReadOnlySpan)
	}{
		{
			name:   "ok",
			status: http.StatusOK,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			// A 4xx is the caller's to judge: e.g. IgnoreNotFound handles a 404.
			name:   "not found",
			status: http.StatusNotFound,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Unset, span.Status().Code)
				for _, a := range span.Attributes() {
					require.NotEqual(t, semconv.ErrorTypeKey, a.Key)
				}
			},
		},
		{
			name:   "conflict",
			status: http.StatusConflict,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Unset, span.Status().Code)
			},
		},
		{
			name:   "server error",
			status: http.StatusInternalServerError,
			assert: func(t *testing.T, span sdktrace.ReadOnlySpan) {
				require.Equal(t, codes.Error, span.Status().Code)
				require.Contains(t, span.Attributes(), semconv.ErrorTypeKey.String("500"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Not parallel: installs a global tracer provider.
			ended := telemetrytest.RecordSpans(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.status)
			}))
			defer srv.Close()
			cfg := &rest.Config{Host: srv.URL}
			WrapRESTConfig(cfg)
			httpClient, err := rest.HTTPClientFor(cfg)
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
			require.NoError(t, err)
			res, err := httpClient.Do(req)
			require.NoError(t, err)
			_ = res.Body.Close()

			span := telemetrytest.EndedSpan(t, ended(), "k8s get freights")
			require.Contains(t, span.Attributes(), semconv.HTTPResponseStatusCode(testCase.status))
			testCase.assert(t, span)
		})
	}
}

// urlErrorRoundTripper fails every request with a *url.Error naming its URL.
type urlErrorRoundTripper struct{}

func (urlErrorRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, &url.Error{Op: r.Method, URL: r.URL.String(), Err: errors.New("connection refused")}
}

func TestWrapRESTConfig_roundTripFails(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	cfg := &rest.Config{Host: "https://k8s.example", Transport: urlErrorRoundTripper{}}
	WrapRESTConfig(cfg)
	httpClient, err := rest.HTTPClientFor(cfg)
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://k8s.example/api/v1/namespaces/smoke/secrets/repo-creds",
		nil,
	)
	require.NoError(t, err)
	_, err = httpClient.Do(req) // nolint: bodyclose
	require.ErrorContains(t, err, "repo-creds")

	span := telemetrytest.EndedSpan(t, ended(), "k8s get secrets")
	require.Equal(t, codes.Error, span.Status().Code)
	require.Contains(t, span.Status().Description, "/secrets/{redacted}")
	require.NotContains(t, span.Status().Description, "repo-creds")
	require.Len(t, span.Events(), 1)
	for _, attr := range span.Events()[0].Attributes {
		require.NotContains(t, attr.Value.String(), "repo-creds")
	}
	for _, attr := range span.Attributes() {
		require.NotContains(t, attr.Value.String(), "repo-creds")
	}
}
