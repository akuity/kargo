package telemetry

import (
	"context"
	"testing"

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
			name: "tracing enabled",
			env:  map[string]string{"TRACING_ENABLED": "true"},
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
