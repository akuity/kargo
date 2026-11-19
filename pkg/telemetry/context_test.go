package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

func TestTracingOff(t *testing.T) {
	// Not parallel: reads global state other tests change.
	ctx := ContextWithAttributes(context.Background(), ProjectKey.String("p"))
	require.Empty(t, AttributesFromContext(ctx))
}

func TestStartSpan(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	ctx := ContextWithAttributes(context.Background(), StageKey.String("dev"), FreightKey.String("a"))
	_, span := StartSpan(ctx, otel.Tracer("test"), "test", trace.WithAttributes(FreightKey.String("b")))
	span.End()
	attrs := ended()[0].Attributes()
	require.Contains(t, attrs, StageKey.String("dev"))
	require.Contains(t, attrs, FreightKey.String("b"))
	require.NotContains(t, attrs, FreightKey.String("a"))
}

func TestStartSpanIn(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	testCases := []struct {
		name       string
		categories string
		assert     func(*testing.T, context.Context, context.Context, trace.Span, bool, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "every category",
			assert: func(
				t *testing.T,
				parentCtx context.Context,
				ctx context.Context,
				span trace.Span,
				recording bool,
				ended []sdktrace.ReadOnlySpan,
			) {
				require.True(t, recording)
				require.Same(t, span, trace.SpanFromContext(ctx))
				require.Len(t, ended, 1)
				require.Equal(t, "child", ended[0].Name())
				require.Contains(t, ended[0].Attributes(), StageKey.String("dev"))
				require.Equal(
					t,
					trace.SpanFromContext(parentCtx).SpanContext().SpanID(),
					ended[0].Parent().SpanID(),
				)
			},
		},
		{
			name:       "category on by default",
			categories: "default",
			assert: func(
				t *testing.T,
				_ context.Context,
				_ context.Context,
				_ trace.Span,
				recording bool,
				ended []sdktrace.ReadOnlySpan,
			) {
				require.True(t, recording)
				require.Len(t, ended, 1)
			},
		},
		{
			name:       "category not selected",
			categories: "none",
			assert: func(
				t *testing.T,
				parentCtx context.Context,
				ctx context.Context,
				span trace.Span,
				recording bool,
				ended []sdktrace.ReadOnlySpan,
			) {
				require.Equal(t, parentCtx, ctx)
				require.False(t, recording)
				// Never the parent: the caller ends what it gets.
				require.NotEqual(t, trace.SpanFromContext(parentCtx), span)
				require.Empty(t, ended)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ended := telemetrytest.RecordSpans(t)
			if testCase.categories != "" {
				telemetrytest.SetCategories(t, testCase.categories)
			}
			parentCtx := ContextWithAttributes(context.Background(), StageKey.String("dev"))
			parentCtx, parent := otel.Tracer("test").Start(parentCtx, "parent")
			ctx, span := StartSpanIn(parentCtx, CategoryDatabase, otel.Tracer("test"), "child")
			recording := span.IsRecording()
			span.SetAttributes(FreightKey.String("a"))
			span.End()
			testCase.assert(t, parentCtx, ctx, span, recording, ended())
			// Nothing touched the parent.
			ro, ok := parent.(sdktrace.ReadOnlySpan)
			require.True(t, ok)
			require.True(t, ro.EndTime().IsZero())
			require.Empty(t, ro.Attributes())
			parent.End()
		})
	}
}

func TestStartSpanIn_tracingOff(t *testing.T) {
	// Not parallel: reads global state other tests change.
	ctx := context.Background()
	got, span := StartSpanIn(ctx, CategoryDatabase, otel.Tracer("test"), "child")
	require.Equal(t, ctx, got)
	require.False(t, span.IsRecording())
	span.End()
}

func TestContextWithLogValues(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	ended := telemetrytest.RecordSpans(t)
	core, logs := observer.New(zap.DebugLevel)
	ctx := logging.ContextWithLogger(context.Background(), logging.Wrap(zap.New(core)))
	ctx = ContextWithAttributes(ctx, StageKey.String("dev"))
	ctx, span := StartSpan(ctx, otel.Tracer("test"), "test")
	ctx = ContextWithLogValues(ctx)
	logging.LoggerFromContext(ctx).Info("hello")
	RecordError(span, errors.New("boom"))
	span.End()

	fields := logs.All()[0].ContextMap()
	require.Equal(t, span.SpanContext().TraceID().String(), fields["trace_id"])
	require.Equal(t, span.SpanContext().SpanID().String(), fields["span_id"])
	// Only the IDs; the logger's own fields name the objects.
	require.NotContains(t, fields, "kargo.stage")
	require.Equal(t, codes.Error, ended()[0].Status().Code)
	require.Equal(t, "boom", ended()[0].Status().Description)
}

func TestContextWithLogValues_unsampled(t *testing.T) {
	// Not parallel: installs a global tracer provider.
	telemetrytest.UseTracerProvider(
		t,
		sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample())),
	)
	ctx, span := otel.Tracer("test").Start(context.Background(), "unsampled")
	defer span.End()
	// An unsampled span is never exported: its IDs would lead nowhere.
	require.Same(
		t,
		logging.LoggerFromContext(ctx),
		logging.LoggerFromContext(ContextWithLogValues(ctx)),
	)
}
