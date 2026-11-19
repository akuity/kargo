package kubeclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/telemetry/telemetrytest"
)

// BenchmarkPatchStatus covers the two paths every reconciler takes through
// PatchStatus: an update that changes nothing (most calls) and one that
// changes a field, on a small and a full Stage status.
func BenchmarkPatchStatus(b *testing.B) {
	scheme := runtime.NewScheme()
	require.NoError(b, kargoapi.AddToScheme(scheme))
	updates := []struct {
		name   string
		update func(*kargoapi.StageStatus)
	}{
		{name: "no-change", update: func(*kargoapi.StageStatus) {}},
		{name: "change", update: func(s *kargoapi.StageStatus) {
			// Toggle, so every call changes one field by the same amount.
			if s.LastHandledRefresh == "r1" {
				s.LastHandledRefresh = "r2"
			} else {
				s.LastHandledRefresh = "r1"
			}
		}},
	}
	for _, historyLen := range []int{1, 10} {
		for _, u := range updates {
			b.Run(fmt.Sprintf("%s/history-%d", u.name, historyLen), func(b *testing.B) {
				stage := benchStage(historyLen)
				c := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(stage.DeepCopy()).
					WithStatusSubresource(&kargoapi.Stage{}).
					Build()
				b.ReportAllocs()
				for b.Loop() {
					if err := PatchStatus(b.Context(), c, stage, u.update); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// benchStage returns a Stage whose status holds historyLen Freight
// collections and the conditions a healthy Stage carries.
func benchStage(historyLen int) *kargoapi.Stage {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-stage"},
		Status: kargoapi.StageStatus{
			FreightSummary:     "ok",
			LastHandledRefresh: "r1",
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Synced"},
				{Type: "Healthy", Status: metav1.ConditionTrue, Reason: "Healthy"},
			},
		},
	}
	for i := range historyLen {
		col := &kargoapi.FreightCollection{}
		col.UpdateOrPush(kargoapi.FreightReference{
			Name:   fmt.Sprintf("freight-%d", i),
			Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "w"},
			Commits: []kargoapi.GitCommit{{
				RepoURL: "https://example.com/repo.git",
				ID:      fmt.Sprintf("%040d", i),
			}},
		})
		stage.Status.FreightHistory.Record(col)
	}
	return stage
}

// BenchmarkPatchStatusTracing measures a status patch with tracing off, and on
// and sampled. Run it with:
//
//	go test -run '^$' -bench PatchStatusTracing -benchtime 20000x -count 5 ./pkg/kubeclient/
//
// Run it without tests, "off" first: this package's tracer binds to the
// first provider installed.
func BenchmarkPatchStatusTracing(b *testing.B) {
	scheme := runtime.NewScheme()
	if err := kargoapi.AddToScheme(scheme); err != nil {
		b.Fatal(err)
	}
	run := func(b *testing.B) {
		freight := &kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: "smoke", Name: "abc"},
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: "wh",
			},
		}
		c := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(freight).
			WithStatusSubresource(freight).
			Build()
		ctx := context.Background()
		var i int
		b.ReportAllocs()
		for b.Loop() {
			// Alternate, so that every patch changes something.
			i++
			if err := PatchStatus(ctx, c, freight, func(s *kargoapi.FreightStatus) {
				if i%2 == 0 {
					s.RemoveCurrentStage("dev")
					return
				}
				s.AddCurrentStage("dev", metav1.Now().Time)
			}); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.Run("off", run)
	b.Run("on-sampled", func(b *testing.B) {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(tracetest.NewNoopExporter()),
		)
		b.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
		telemetrytest.UseTracerProvider(b, tp)
		run(b)
	})
}
