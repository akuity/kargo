package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/telemetry/internal/state"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

// The benchmarks below measure what tracing costs a reconcile. Run them with:
//
//	go test -run '^$' -bench . -benchtime 200000x -count 5 ./pkg/telemetry/

// benchMode is a tracing state a benchmark runs in.
type benchMode struct {
	name     string
	enabled  bool
	provider trace.TracerProvider
}

// tracer returns a tracer from m's provider.
func (m benchMode) tracer() trace.Tracer {
	return m.provider.Tracer("bench")
}

// benchModes returns tracing off, on but unsampled, and on and sampled.
func benchModes(b *testing.B) []benchMode {
	b.Helper()
	unsampled := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)
	b.Cleanup(func() { _ = unsampled.Shutdown(context.Background()) })
	return []benchMode{
		{name: "off", provider: noop.NewTracerProvider()},
		{name: "on-unsampled", enabled: true, provider: unsampled},
		sampledBenchMode(b),
	}
}

// sampledBenchMode returns tracing on and sampled, with spans exported to
// nowhere.
func sampledBenchMode(b *testing.B) benchMode {
	b.Helper()
	sampled := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tracetest.NewNoopExporter()),
	)
	b.Cleanup(func() { _ = sampled.Shutdown(context.Background()) })
	return benchMode{name: "on-sampled", enabled: true, provider: sampled}
}

// useBenchMode installs m's provider globally and turns tracing on or off for
// the rest of the benchmark.
func useBenchMode(b *testing.B, m benchMode) {
	b.Helper()
	telemetrytest.UseTracerProvider(b, m.provider)
	state.Enabled.Store(m.enabled)
}

// baselineReconcileSpan is a reconcile's span without the helpers.
func baselineReconcileSpan(ctx context.Context, tracer trace.Tracer) {
	_, span := tracer.Start(
		ctx,
		"Reconcile Stage",
		trace.WithAttributes(ProjectKey.String("smoke"), StageKey.String("dev")),
	)
	span.End()
}

// instrumentedReconcileSpan is a reconcile's span with the helpers, as the
// reconcilers use them.
func instrumentedReconcileSpan(ctx context.Context, tracer trace.Tracer) {
	ctx = ContextWithAttributes(ctx, ProjectKey.String("smoke"), StageKey.String("dev"))
	ctx, span := StartSpan(ctx, tracer, "Reconcile Stage")
	_ = ContextWithLogValues(ctx)
	RecordError(span, nil)
	span.End()
}

// promotionAttributesSpan is a span with the Stage and Freight attributes a
// Promotion reconcile adds.
func promotionAttributesSpan(
	ctx context.Context,
	tracer trace.Tracer,
	freight *kargoapi.Freight,
) {
	ctx, span := StartSpan(ctx, tracer, "Reconcile Promotion")
	if Enabled() {
		attrs := []attribute.KeyValue{
			StageKey.String("dev"),
			FreightKey.String(freight.Name),
		}
		span.SetAttributes(attrs...)
		ctx = ContextWithAttributes(ctx, attrs...)
		attrs = FreightDetailAttributes(freight)
		span.SetAttributes(attrs...)
		_ = ContextWithAttributes(ctx, attrs...)
	}
	span.End()
}

func BenchmarkReconcileSpan(b *testing.B) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc123"},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "wh",
		},
		Commits: []kargoapi.GitCommit{{ID: "deadbeef"}},
	}
	ctx := context.Background()
	for _, m := range benchModes(b) {
		b.Run("baseline/"+m.name, func(b *testing.B) {
			useBenchMode(b, m)
			b.ReportAllocs()
			for b.Loop() {
				baselineReconcileSpan(ctx, m.tracer())
			}
		})
		b.Run("instrumented/"+m.name, func(b *testing.B) {
			useBenchMode(b, m)
			b.ReportAllocs()
			for b.Loop() {
				instrumentedReconcileSpan(ctx, m.tracer())
			}
		})
		b.Run("promotion-attributes/"+m.name, func(b *testing.B) {
			useBenchMode(b, m)
			b.ReportAllocs()
			for b.Loop() {
				promotionAttributesSpan(ctx, m.tracer(), freight)
			}
		})
	}
}

// BenchmarkHelpers measures each helper with tracing on and sampled.
func BenchmarkHelpers(b *testing.B) {
	m := sampledBenchMode(b)
	useBenchMode(b, m)
	ctx := ContextWithAttributes(
		context.Background(),
		ProjectKey.String("smoke"),
		StageKey.String("dev"),
	)
	spanCtx, span := m.tracer().Start(ctx, "parent")
	defer span.End()
	b.Run("ContextWithAttributes", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ContextWithAttributes(
				context.Background(),
				ProjectKey.String("smoke"),
				StageKey.String("dev"),
			)
		}
	})
	b.Run("ContextWithLogValues", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ContextWithLogValues(spanCtx)
		}
	})
	b.Run("StartSpan", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, s := StartSpan(ctx, m.tracer(), "child")
			s.End()
		}
	})
	b.Run("tracer.Start", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, s := m.tracer().Start(ctx, "child")
			s.End()
		}
	})
}

// BenchmarkReference measures work a reconcile already does, for scale.
func BenchmarkReference(b *testing.B) {
	scheme := runtime.NewScheme()
	if err := kargoapi.AddToScheme(scheme); err != nil {
		b.Fatal(err)
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "dev"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage).Build()
	b.Run("fake client Get", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = c.Get(
				context.Background(),
				client.ObjectKeyFromObject(stage),
				&kargoapi.Stage{},
			)
		}
	})
	logger := logging.LoggerFromContext(context.Background())
	b.Run("logger.WithValues", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = logger.WithValues("namespace", "smoke", "stage", "dev")
		}
	})
}
