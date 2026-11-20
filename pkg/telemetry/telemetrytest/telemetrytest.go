// Package telemetrytest provides helpers for testing code instrumented with
// package telemetry.
package telemetrytest

import (
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/akuity/kargo/pkg/telemetry/internal/state"
)

// recorder and provider are shared per test binary: a package-level tracer
// binds to the first global provider installed.
var (
	recorder = tracetest.NewSpanRecorder()
	provider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
)

// UseTracerProvider installs tp globally and turns tracing on, every category
// included, until tb ends. Not for parallel tests.
func UseTracerProvider(tb testing.TB, tp trace.TracerProvider) {
	tb.Helper()
	prevProvider := otel.GetTracerProvider()
	prevPropagator := otel.GetTextMapPropagator()
	prevEnabled := state.Enabled.Load()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	state.Enabled.Store(true)
	SetCategories(tb, state.AllCategories)
	tb.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevPropagator)
		state.Enabled.Store(prevEnabled)
	})
}

// SetCategories selects tracing categories as TRACING_CATEGORIES=value would,
// until tb ends. Names are not checked. Not for parallel tests.
func SetCategories(tb testing.TB, value string) {
	tb.Helper()
	selection, err := state.ParseCategories(strings.Split(value, ","))
	if err != nil {
		tb.Fatalf("error parsing tracing categories %q: %v", value, err)
	}
	prev := state.Selected.Load()
	state.Selected.Store(selection)
	tb.Cleanup(func() { state.Selected.Store(prev) })
}

// RecordSpans installs a recording tracer provider until tb ends and returns
// a function that returns the spans ended since.
func RecordSpans(tb testing.TB) func() []sdktrace.ReadOnlySpan {
	tb.Helper()
	UseTracerProvider(tb, provider)
	n := len(recorder.Ended())
	return func() []sdktrace.ReadOnlySpan { return recorder.Ended()[n:] }
}

// EndedSpan returns the last of spans named name, failing tb if there is
// none.
func EndedSpan(
	tb testing.TB,
	spans []sdktrace.ReadOnlySpan,
	name string,
) sdktrace.ReadOnlySpan {
	tb.Helper()
	for i := len(spans) - 1; i >= 0; i-- {
		if spans[i].Name() == name {
			return spans[i]
		}
	}
	tb.Fatalf("no span named %q", name)
	return nil
}
