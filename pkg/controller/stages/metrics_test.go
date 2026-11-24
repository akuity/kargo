package stages

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient/testutil"
	"github.com/akuity/kargo/pkg/controller/metrics"
)

func TestSyncFreight_metrics(t *testing.T) {
	t.Parallel()

	// The histogram is process-wide; a unique Project isolates this test.
	const project = "fake-project-sync-freight-metrics"
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	verified := metav1.Time{Time: created.Add(90 * time.Second)}

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         project,
			Name:              "test-freight",
			CreationTimestamp: metav1.Time{Time: created},
		},
	}
	// verifiedFreight is the same Freight once the Stage has recorded it as
	// verified; SyncFreight observes the lead time only on the transition.
	verifiedFreight := freight.DeepCopy()
	verifiedFreight.Status.VerifiedIn =
		map[string]kargoapi.VerifiedStage{"test-stage": {}}

	// getFreight returns whichever Freight the test currently wants GetFreight to
	// yield, letting the test drive the not-yet-verified -> verified transition
	// without the mock mutating state.
	getFreight := freight
	mockClient := &testutil.MockFreightClient{
		GetFreightFun: func(context.Context, string, string) (*kargoapi.Freight, error) {
			return getFreight, nil
		},
		MarkFreightsCurrentlyInStageFun: func(context.Context, string, string, []string) error {
			return nil
		},
		MarkFreightsVerifiedInStageFun: func(
			context.Context, string, string, []string, time.Time,
		) error {
			return nil
		},
	}
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

	require.NoError(t, SyncFreight(t.Context(), mockClient, stage))
	require.Equal(t, uint64(1), sample().GetSampleCount())
	require.Equal(t, 90.0, sample().GetSampleSum())

	// Freight already verified in the Stage is not observed again.
	getFreight = verifiedFreight
	require.NoError(t, SyncFreight(t.Context(), mockClient, stage))
	require.Equal(t, uint64(1), sample().GetSampleCount())
}
