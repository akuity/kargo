package external

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestServer_Serve_tracing(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
	))

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))
	// A client that fails every list makes the route handler answer quickly
	// with a server error, which is all this test needs of it.
	testClient := fake.NewClientBuilder().WithScheme(testScheme).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(
				context.Context,
				client.WithWatch,
				client.ObjectList,
				...client.ListOption,
			) error {
				return errors.New("something went wrong")
			},
		}).Build()

	s, ok := NewServer(
		ServerConfig{TracingEnabled: true},
		testClient,
		testClient,
	).(*server)
	require.True(t, ok)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		// The error is irrelevant: canceling the context stops the server.
		_ = s.Serve(ctx, l)
	}()

	get := func(path string) int {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			fmt.Sprintf("http://%s%s", l.Addr().String(), path),
			nil,
		)
		require.NoError(t, err)
		// Retry a few times in case the server isn't quite ready yet.
		var resp *http.Response
		for range 3 {
			if resp, err = http.DefaultClient.Do(req); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.StatusCode
	}

	// Health checks are not traced.
	require.Equal(t, http.StatusOK, get("/healthz"))
	require.Empty(t, recorder.Ended())

	// Everything else is.
	require.Equal(t, http.StatusInternalServerError, get("/some/receiver"))
	ended := recorder.Ended()
	require.Len(t, ended, 1)
	span := ended[0]
	require.Equal(t, webhookSpanName, span.Name())
	require.Equal(t, trace.SpanKindServer, span.SpanKind())
	require.Subset(
		t,
		span.Attributes(),
		[]any{
			semconv.HTTPRequestMethodGet,
			semconv.URLPath("/some/receiver"),
			semconv.HTTPResponseStatusCode(http.StatusInternalServerError),
		},
	)
}
