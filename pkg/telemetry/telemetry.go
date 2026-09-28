// Package telemetry wires the OpenTelemetry SDK into Kargo's long-running
// components. Setup is called once at startup by each component; code that
// records spans uses the OpenTelemetry API directly and works whether or not
// Setup was ever called, since the API's no-op implementations are in effect
// until it is.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/x/version"
)

// shutdownTimeout bounds how long a component waits at shutdown for spans not
// yet exported to be flushed.
const shutdownTimeout = 10 * time.Second

// Service describes the component whose telemetry is being set up. Its fields
// become resource attributes on every span the component exports.
type Service struct {
	// Name is the logical name of the component, e.g. "kargo-controller". It
	// becomes the service.name resource attribute unless OTEL_SERVICE_NAME
	// overrides it.
	Name string
	// Version is the version of Kargo the component is running.
	Version string
	// Attributes are any additional resource attributes to attach.
	Attributes []attribute.KeyValue
}

// ShutdownFunc flushes any telemetry not yet exported and releases the
// resources held by the SDK. It should be called once, at shutdown.
type ShutdownFunc func(context.Context) error

// Setup configures the global OpenTelemetry tracer provider and propagators
// according to cfg and returns a ShutdownFunc. When tracing is disabled, the
// global no-op implementations are left in place and the returned ShutdownFunc
// does nothing.
//
// The exporter is configured from the standard OTEL_EXPORTER_OTLP_* environment
// variables, the sampler from OTEL_TRACES_SAMPLER and OTEL_TRACES_SAMPLER_ARG,
// and additional resource attributes from OTEL_RESOURCE_ATTRIBUTES.
func Setup(ctx context.Context, cfg Config, svc Service) (ShutdownFunc, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	exporter, err := newTraceExporter(ctx, cfg.EffectiveTracesProtocol())
	if err != nil {
		return nil, fmt.Errorf("error creating trace exporter: %w", err)
	}

	res, err := newResource(ctx, svc)
	if err != nil {
		return nil, fmt.Errorf("error building telemetry resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	logger := logging.LoggerFromContext(ctx).WithoutStackTraces()
	otel.SetLogger(logger.Logr())
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Error(err, "OpenTelemetry error")
	}))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tp.Shutdown, nil
}

// SetupFromEnv configures tracing for the named Kargo component (for example
// "controller", which is reported as the service "kargo-controller") according
// to the environment, and returns a function that flushes and shuts tracing
// down. That function is meant to be deferred by the component's entry point:
// it never fails, and a problem flushing at shutdown is logged instead. When
// tracing is disabled, both this function and the one it returns are no-ops.
//
// Any attrs are attached to the component's resource, in addition to its
// service name and version.
func SetupFromEnv(
	ctx context.Context,
	component string,
	attrs ...attribute.KeyValue,
) (func(), error) {
	cfg := ConfigFromEnv()
	shutdown, err := Setup(
		ctx,
		cfg,
		Service{
			Name:       "kargo-" + component,
			Version:    version.GetVersion().Version,
			Attributes: attrs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing tracing: %w", err)
	}
	logger := logging.LoggerFromContext(ctx)
	if cfg.Enabled {
		logger.Info(
			"tracing is enabled",
			"protocol", cfg.EffectiveTracesProtocol(),
		)
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			logger.Error(err, "error shutting down tracing")
		}
	}, nil
}

// newTraceExporter returns a span exporter that speaks the given OTLP
// protocol. The exporter reads its endpoint, headers, TLS settings, and so on
// from the standard OTEL_EXPORTER_OTLP_* environment variables.
func newTraceExporter(
	ctx context.Context,
	protocol Protocol,
) (sdktrace.SpanExporter, error) {
	switch protocol {
	case ProtocolGRPC:
		return otlptracegrpc.New(ctx)
	case ProtocolHTTPProtobuf:
		return otlptracehttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol %q", protocol)
	}
}

// newResource describes the component to the telemetry backend. Attributes
// from the environment (OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES) are
// applied last so that they override the defaults derived from svc.
func newResource(ctx context.Context, svc Service) (*resource.Resource, error) {
	return resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(svc.Name),
			semconv.ServiceVersion(svc.Version),
		),
		resource.WithAttributes(svc.Attributes...),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
}
