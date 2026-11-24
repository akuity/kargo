package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/akuity/kargo/pkg/logging"
)

type attributesKey struct{}

// ContextWithAttributes returns ctx carrying attrs, added to any it already
// carries, for StartSpan. A no-op when tracing is off.
func ContextWithAttributes(
	ctx context.Context,
	attrs ...attribute.KeyValue,
) context.Context {
	if !Enabled() || len(attrs) == 0 {
		return ctx
	}
	prev := AttributesFromContext(ctx)
	all := make([]attribute.KeyValue, 0, len(prev)+len(attrs))
	all = append(all, prev...)
	all = append(all, attrs...)
	return context.WithValue(ctx, attributesKey{}, all)
}

// AttributesFromContext returns the attributes ContextWithAttributes put in
// ctx.
func AttributesFromContext(ctx context.Context) []attribute.KeyValue {
	attrs, _ := ctx.Value(attributesKey{}).([]attribute.KeyValue)
	return attrs
}

// StartSpan starts a span with the attributes carried by ctx, followed by any
// given in opts, so that the latter win.
func StartSpan(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	if attrs := AttributesFromContext(ctx); len(attrs) > 0 {
		opts = append([]trace.SpanStartOption{trace.WithAttributes(attrs...)}, opts...)
	}
	return tracer.Start(ctx, name, opts...)
}

// StartSpanIn is StartSpan for a span in category c. When c isn't enabled,
// it returns ctx unchanged and a non-recording span, never ctx's own, so the
// caller may end it.
func StartSpanIn(
	ctx context.Context,
	c Category,
	tracer trace.Tracer,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	if !CategoryEnabled(c) {
		return ctx, noop.Span{}
	}
	return StartSpan(ctx, tracer, name, opts...)
}

// ContextWithLogValues returns ctx with its logger carrying the span's trace
// and span IDs, if sampled. Call it once, at the root of a unit of work:
// calling it again repeats the fields.
func ContextWithLogValues(ctx context.Context) context.Context {
	sc := trace.SpanContextFromContext(ctx)
	if !Enabled() || !sc.IsSampled() {
		return ctx
	}
	return logging.ContextWithLogger(
		ctx,
		logging.LoggerFromContext(ctx).WithValues(
			"trace_id", sc.TraceID().String(),
			"span_id", sc.SpanID().String(),
		),
	)
}
