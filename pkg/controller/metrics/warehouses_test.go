package metrics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_oldestDiscoveryByProject(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	warehouse := func(project string, discovered *time.Time) kargoapi.Warehouse {
		wh := kargoapi.Warehouse{ObjectMeta: metav1.ObjectMeta{Namespace: project}}
		if discovered != nil {
			wh.Status.DiscoveredArtifacts = &kargoapi.DiscoveredArtifacts{
				DiscoveredAt: metav1.Time{Time: *discovered},
			}
		}
		return wh
	}
	at := func(ago time.Duration) *time.Time {
		t := t0.Add(-ago)
		return &t
	}

	require.Empty(t, oldestDiscoveryByProject(nil))
	require.Equal(
		t,
		map[string]time.Time{
			"project-a": t0.Add(-10 * time.Minute),
			"project-b": t0.Add(-time.Minute),
		},
		oldestDiscoveryByProject([]kargoapi.Warehouse{
			warehouse("project-a", at(time.Minute)),
			warehouse("project-a", at(10*time.Minute)),
			// Never discovered: not considered.
			warehouse("project-a", nil),
			warehouse("project-b", at(time.Minute)),
			warehouse("project-c", nil),
		}),
	)
}

func TestNewWarehouseMetrics(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := NewWarehouseMetrics(c)
	require.Same(t, m, NewWarehouseMetrics(nil))
	m.DiscoveryDuration.WithLabelValues("project-a").Observe(1)
	m.DiscoveryInterval.WithLabelValues("project-a").Observe(60)
	m.FreightCreated.WithLabelValues("project-a").Inc()
	count, err := testutil.GatherAndCount(
		ctrlmetrics.Registry,
		"kargo_warehouse_discovery_duration_seconds",
		"kargo_warehouse_discovery_interval_seconds",
		"kargo_warehouse_freight_created_total",
	)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func Test_warehouseDiscoveryCollector(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{Namespace: "project-a", Name: "wh"},
		Status: kargoapi.WarehouseStatus{
			DiscoveredArtifacts: &kargoapi.DiscoveredArtifacts{
				DiscoveredAt: metav1.Time{Time: t0},
			},
		},
	}
	testCases := []struct {
		name     string
		client   client.Client
		expected string
	}{
		{
			name: "error listing Warehouses",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return errors.New("something went wrong")
					},
				}).
				Build(),
		},
		{
			name: "success",
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(warehouse).
				WithStatusSubresource(warehouse).
				Build(),
			// nolint: lll
			expected: `
# HELP kargo_warehouse_oldest_discovery_timestamp_seconds Time, as a Unix timestamp, of the least recent successful artifact discovery among the Project's Warehouses. Warehouses that have never discovered artifacts are not considered.
# TYPE kargo_warehouse_oldest_discovery_timestamp_seconds gauge
kargo_warehouse_oldest_discovery_timestamp_seconds{project="project-a"} 1.7672256e+09
`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, testutil.CollectAndCompare(
				newWarehouseDiscoveryCollector(testCase.client),
				strings.NewReader(testCase.expected),
			))
		})
	}
}
