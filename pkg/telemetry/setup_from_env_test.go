package telemetry

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestSetupFromEnv(t *testing.T) {
	// Not parallel: manipulates the environment and global state.
	testCases := []struct {
		name   string
		env    map[string]string
		assert func(*testing.T, func(), error)
	}{
		{
			name: "tracing disabled",
			assert: func(t *testing.T, shutdown func(), err error) {
				require.NoError(t, err)
				require.NotNil(t, shutdown)
				// The global provider must not have been replaced...
				_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
				require.False(t, ok)
				// ...and shutting down must be harmless.
				shutdown()
			},
		},
		{
			name: "unsupported protocol",
			env: map[string]string{
				"TRACING_ENABLED":             "true",
				"OTEL_EXPORTER_OTLP_PROTOCOL": "http/json",
			},
			assert: func(t *testing.T, shutdown func(), err error) {
				require.ErrorContains(t, err, "error initializing tracing")
				require.ErrorContains(t, err, "unsupported OTLP protocol")
				require.Nil(t, shutdown)
			},
		},
		{
			name: "unknown category",
			env: map[string]string{
				"TRACING_ENABLED":    "true",
				"TRACING_CATEGORIES": "database,bogus",
			},
			assert: func(t *testing.T, shutdown func(), err error) {
				require.ErrorContains(t, err, "error initializing tracing")
				require.ErrorContains(t, err, `unknown tracing category "bogus"`)
				require.Nil(t, shutdown)
			},
		},
		{
			name: "unknown category with tracing off",
			env:  map[string]string{"TRACING_CATEGORIES": "bogus"},
			assert: func(t *testing.T, shutdown func(), err error) {
				require.NoError(t, err)
				require.NotNil(t, shutdown)
				shutdown()
			},
		},
		{
			name: "categories set but empty",
			env: map[string]string{
				"TRACING_ENABLED":            "true",
				"TRACING_CATEGORIES":         "",
				"OTEL_EXPORTER_OTLP_TIMEOUT": "100",
			},
			assert: func(t *testing.T, shutdown func(), err error) {
				require.NoError(t, err)
				require.True(t, CategoryEnabled(CategoryDatabase))
				shutdown()
			},
		},
		{
			name: "tracing enabled",
			env:  map[string]string{"TRACING_ENABLED": "true", "OTEL_EXPORTER_OTLP_TIMEOUT": "100"},
			assert: func(t *testing.T, shutdown func(), err error) {
				require.NoError(t, err)
				require.NotNil(t, shutdown)
				tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
				require.True(t, ok)
				// A span from the installed provider reveals the resource the
				// component was described with.
				_, span := tp.Tracer("test").Start(context.Background(), "test")
				roSpan, ok := span.(sdktrace.ReadOnlySpan)
				require.True(t, ok)
				attrs := roSpan.Resource().Set()
				val, _ := attrs.Value(semconv.ServiceNameKey)
				require.Equal(t, "kargo-test", val.AsString())
				_, ok = attrs.Value(semconv.ServiceVersionKey)
				require.True(t, ok)
				val, _ = attrs.Value(ShardKey)
				require.Equal(t, "east", val.AsString())
				span.End()
				// Shutting down must flush without panicking, even with nowhere
				// to export to.
				shutdown()
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resetGlobals(t)
			for k, v := range testCase.env {
				t.Setenv(k, v)
			}
			shutdown, err := SetupFromEnv(
				context.Background(),
				"test",
				ShardKey.String("east"),
			)
			testCase.assert(t, shutdown, err)
		})
	}
}

func TestSetupFromEnv_collectorDown(t *testing.T) {
	// Not parallel: manipulates the environment and global state.
	resetGlobals(t)
	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+hangingListener(t))
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "100")
	t.Setenv("OTEL_BSP_MAX_QUEUE_SIZE", "8")
	t.Setenv("OTEL_BSP_MAX_EXPORT_BATCH_SIZE", "4")
	shutdown, err := SetupFromEnv(context.Background(), "test")
	require.NoError(t, err)

	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, ok)
	tracer := tp.Tracer("test")

	// Ending spans must not block on a collector that never responds.
	requireReturns(t, 10*time.Second, func() {
		for range 10_000 {
			_, span := tracer.Start(context.Background(), "test")
			span.End()
		}
	})
	requireReturns(t, shutdownTimeout+10*time.Second, shutdown)
}

// hangingListener returns the address of a listener that accepts connections
// and never responds.
func hangingListener(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var (
		mu    sync.Mutex
		conns []net.Conn
	)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		require.NoError(t, l.Close())
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			if err := c.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	return l.Addr().String()
}

// requireReturns fails t if f doesn't return within timeout.
func requireReturns(t *testing.T, timeout time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("did not return within %s", timeout)
	}
}
