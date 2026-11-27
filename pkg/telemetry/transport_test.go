package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

func TestWrapTransport_tracingOff(t *testing.T) {
	// Not parallel: reads global state other tests change.
	rt := okRoundTripper{}
	require.Equal(t, http.RoundTripper(rt), WrapTransport(rt, "registry"))
}

func TestWrapTransport_categoryNotEnabled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	telemetrytest.RecordSpans(t)
	// discovery is off by default.
	telemetrytest.SetCategories(t, "default")
	rt := okRoundTripper{}
	require.Equal(t, http.RoundTripper(rt), WrapTransport(rt, "registry"))
}

func TestWrapTransport(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	var mu sync.Mutex
	var traceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		traceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := ContextWithAttributes(context.Background(), WarehouseKey.String("wh"))
	ctx, parent := StartSpan(ctx, otel.Tracer("test"), "parent")
	c := &http.Client{Transport: WrapTransport(http.DefaultTransport, "registry")}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		srv.URL+"/v2/blob?X-Amz-Signature=abc",
		nil,
	)
	require.NoError(t, err)
	res, err := c.Do(req)
	require.NoError(t, err)
	_ = res.Body.Close()
	parent.End()

	// Third parties get no trace context, and url.full has no query.
	mu.Lock()
	require.Empty(t, traceparent)
	mu.Unlock()
	span := telemetrytest.EndedSpan(t, ended(), "registry GET")
	require.Equal(t, trace.SpanKindClient, span.SpanKind())
	require.Contains(t, span.Attributes(), semconv.URLFull(srv.URL+"/v2/blob"))
	require.Contains(t, span.Attributes(), WarehouseKey.String("wh"))
}

func TestWrapTransport_notFound(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	// Registries answer probes with a 404, which callers handle.
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	c := &http.Client{Transport: WrapTransport(http.DefaultTransport, "registry")}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodHead, srv.URL+"/v2/app/manifests/v1", nil)
	require.NoError(t, err)
	res, err := c.Do(req)
	require.NoError(t, err)
	_ = res.Body.Close()

	span := telemetrytest.EndedSpan(t, ended(), "registry HEAD")
	require.Equal(t, codes.Unset, span.Status().Code)
	require.Contains(t, span.Attributes(), semconv.HTTPResponseStatusCode(http.StatusNotFound))
}
