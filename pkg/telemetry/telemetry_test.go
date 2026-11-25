package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// resetGlobals restores the global OpenTelemetry state after a test that
// called Setup, so that tests do not leak a tracer provider into one another.
func resetGlobals(t *testing.T) {
	t.Helper()
	prevProvider := otel.GetTracerProvider()
	prevPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevPropagator)
	})
}

func TestSetup(t *testing.T) {
	// Not parallel: Setup mutates global state.
	testCases := []struct {
		name   string
		cfg    Config
		assert func(*testing.T, ShutdownFunc, error)
	}{
		{
			name: "disabled",
			cfg:  Config{Enabled: false},
			assert: func(t *testing.T, shutdown ShutdownFunc, err error) {
				require.NoError(t, err)
				require.NotNil(t, shutdown)
				require.NoError(t, shutdown(context.Background()))
				// The global provider must not have been replaced.
				_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
				require.False(t, ok)
			},
		},
		{
			name: "unsupported protocol",
			cfg:  Config{Enabled: true, Protocol: "http/json"},
			assert: func(t *testing.T, shutdown ShutdownFunc, err error) {
				require.ErrorContains(t, err, "unsupported OTLP protocol")
				require.Nil(t, shutdown)
			},
		},
		{
			name: "enabled over gRPC",
			cfg:  Config{Enabled: true, Protocol: ProtocolGRPC},
			assert: func(t *testing.T, shutdown ShutdownFunc, err error) {
				require.NoError(t, err)
				require.NotNil(t, shutdown)
				_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
				require.True(t, ok)
				require.ElementsMatch(
					t,
					[]string{"traceparent", "tracestate", "baggage"},
					otel.GetTextMapPropagator().Fields(),
				)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				require.NoError(t, shutdown(ctx))
			},
		},
		{
			name: "enabled over HTTP",
			cfg:  Config{Enabled: true, Protocol: ProtocolHTTPProtobuf},
			assert: func(t *testing.T, shutdown ShutdownFunc, err error) {
				require.NoError(t, err)
				require.NotNil(t, shutdown)
				_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
				require.True(t, ok)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				require.NoError(t, shutdown(ctx))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resetGlobals(t)
			shutdown, err := Setup(
				context.Background(),
				testCase.cfg,
				Service{Name: "kargo-test", Version: "v0.0.0"},
			)
			testCase.assert(t, shutdown, err)
		})
	}
}

func TestNewTraceExporter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		protocol Protocol
		assert   func(*testing.T, sdktrace.SpanExporter, error)
	}{
		{
			name:     "grpc",
			protocol: ProtocolGRPC,
			assert: func(t *testing.T, exporter sdktrace.SpanExporter, err error) {
				require.NoError(t, err)
				require.NotNil(t, exporter)
				require.NoError(t, exporter.Shutdown(context.Background()))
			},
		},
		{
			name:     "http/protobuf",
			protocol: ProtocolHTTPProtobuf,
			assert: func(t *testing.T, exporter sdktrace.SpanExporter, err error) {
				require.NoError(t, err)
				require.NotNil(t, exporter)
				require.NoError(t, exporter.Shutdown(context.Background()))
			},
		},
		{
			name:     "unsupported",
			protocol: "carrier-pigeon",
			assert: func(t *testing.T, exporter sdktrace.SpanExporter, err error) {
				require.ErrorContains(t, err, `unsupported OTLP protocol "carrier-pigeon"`)
				require.Nil(t, exporter)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			exporter, err := newTraceExporter(context.Background(), testCase.protocol)
			testCase.assert(t, exporter, err)
		})
	}
}

func TestNewResource(t *testing.T) {
	// Not parallel: one case manipulates the environment.
	testCases := []struct {
		name   string
		env    map[string]string
		svc    Service
		assert func(*testing.T, *attribute.Set)
	}{
		{
			name: "service described by the caller",
			svc: Service{
				Name:       "kargo-controller",
				Version:    "v1.2.3",
				Attributes: []attribute.KeyValue{ShardKey.String("east")},
			},
			assert: func(t *testing.T, attrs *attribute.Set) {
				val, _ := attrs.Value(semconv.ServiceNameKey)
				require.Equal(t, "kargo-controller", val.AsString())
				val, _ = attrs.Value(semconv.ServiceVersionKey)
				require.Equal(t, "v1.2.3", val.AsString())
				val, _ = attrs.Value(ShardKey)
				require.Equal(t, "east", val.AsString())
				// The SDK describes itself as well.
				_, ok := attrs.Value(semconv.TelemetrySDKNameKey)
				require.True(t, ok)
			},
		},
		{
			name: "environment overrides the caller",
			env: map[string]string{
				"OTEL_SERVICE_NAME":        "custom-name",
				"OTEL_RESOURCE_ATTRIBUTES": "kargo.shard=west,deployment.environment=prod",
			},
			svc: Service{
				Name:       "kargo-controller",
				Attributes: []attribute.KeyValue{ShardKey.String("east")},
			},
			assert: func(t *testing.T, attrs *attribute.Set) {
				val, _ := attrs.Value(semconv.ServiceNameKey)
				require.Equal(t, "custom-name", val.AsString())
				val, _ = attrs.Value(ShardKey)
				require.Equal(t, "west", val.AsString())
				val, _ = attrs.Value(attribute.Key("deployment.environment"))
				require.Equal(t, "prod", val.AsString())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for k, v := range testCase.env {
				t.Setenv(k, v)
			}
			res, err := newResource(context.Background(), testCase.svc)
			require.NoError(t, err)
			testCase.assert(t, res.Set())
		})
	}
}
