package warehouses

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/metrics"
)

func TestSyncWarehouse_metrics(t *testing.T) {
	t.Parallel()

	histogram := func(name string) *prometheus.HistogramVec {
		return prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, []string{"project"})
	}
	// WarehouseMetrics is a process-wide singleton; use standalone collectors.
	m := &metrics.WarehouseMetrics{
		DiscoveryDuration: histogram("test_discovery_duration_seconds"),
		DiscoveryInterval: histogram("test_discovery_interval_seconds"),
		FreightCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "test_freight_created_total"},
			[]string{"project"},
		),
	}
	discoveredAt := metav1.Now()
	r := &reconciler{
		whMetrics: m,
		discoverArtifactsFn: func(
			context.Context, string,
			[]kargoapi.RepoSubscription,
			*kargoapi.DiscoveredArtifacts,
		) (*kargoapi.DiscoveredArtifacts, error) {
			return &kargoapi.DiscoveredArtifacts{
				DiscoveredAt: discoveredAt,
				Git: []kargoapi.GitDiscoveryResult{
					{RepoURL: "fake-repo", Commits: []kargoapi.DiscoveredCommit{{ID: "fake-commit"}}},
				},
			}, nil
		},
		buildFreightFromLatestArtifactsFn: func(
			string,
			*kargoapi.DiscoveredArtifacts,
		) (*kargoapi.Freight, error) {
			return &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Name: "fake-freight", Namespace: "fake-project"},
				Commits:    []kargoapi.GitCommit{{ID: "fake-commit"}},
			}, nil
		},
		createFreightFn: func(context.Context, client.Object, ...client.CreateOption) error {
			return nil
		},
		patchStatusFn: func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error {
			return nil
		},
	}
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-warehouse", Generation: 1},
		Spec: kargoapi.WarehouseSpec{
			FreightCreationPolicy: kargoapi.FreightCreationPolicyAutomatic,
			Interval:              metav1.Duration{Duration: time.Minute},
		},
		Status: kargoapi.WarehouseStatus{
			ObservedGeneration: 1,
			DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
				DiscoveredAt: metav1.Time{Time: discoveredAt.Add(-5 * time.Minute)},
			},
		},
	}

	_, err := r.syncWarehouse(t.Context(), warehouse)
	require.NoError(t, err)

	histogramOf := func(vec *prometheus.HistogramVec) *dto.Histogram {
		observer, ok := vec.WithLabelValues("fake-project").(prometheus.Metric)
		require.True(t, ok)
		var out dto.Metric
		require.NoError(t, observer.Write(&out))
		return out.GetHistogram()
	}
	require.Equal(t, uint64(1), histogramOf(m.DiscoveryDuration).GetSampleCount())
	interval := histogramOf(m.DiscoveryInterval)
	require.Equal(t, uint64(1), interval.GetSampleCount())
	require.InDelta(t, 300, interval.GetSampleSum(), 0.001)

	var created dto.Metric
	require.NoError(t, m.FreightCreated.WithLabelValues("fake-project").Write(&created))
	require.Equal(t, 1.0, created.GetCounter().GetValue())
}
