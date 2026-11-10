package stages

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/metrics"
	"github.com/akuity/kargo/pkg/indexer"
)

func TestSyncFreight_metrics(t *testing.T) {
	t.Parallel()

	// The histogram is process-wide; a unique Project isolates this test.
	const project = "fake-project-sync-freight-metrics"
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	verified := metav1.Time{Time: created.Add(90 * time.Second)}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         project,
			Name:              "test-freight",
			CreationTimestamp: metav1.Time{Time: created},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(freight).WithStatusSubresource(freight).
		WithIndex(&kargoapi.Freight{}, indexer.FreightByCurrentStagesField, indexer.FreightByCurrentStages).
		Build()
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Namespace: project, Name: "test-stage"},
		Status: kargoapi.StageStatus{
			Health: &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
			FreightHistory: kargoapi.FreightHistory{{
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{{
					Phase:      kargoapi.VerificationPhaseSuccessful,
					FinishTime: &verified,
				}},
			}},
		},
	}

	sample := func() *dto.Histogram {
		observer, ok := metrics.NewFreightMetrics().VerificationLeadTime.
			WithLabelValues(project).(prometheus.Metric)
		require.True(t, ok)
		var m dto.Metric
		require.NoError(t, observer.Write(&m))
		return m.GetHistogram()
	}

	require.NoError(t, SyncFreight(t.Context(), c, stage))
	require.Equal(t, uint64(1), sample().GetSampleCount())
	require.Equal(t, 90.0, sample().GetSampleSum())

	// Freight already verified in the Stage is not observed again.
	require.NoError(t, SyncFreight(t.Context(), c, stage))
	require.Equal(t, uint64(1), sample().GetSampleCount())
}
