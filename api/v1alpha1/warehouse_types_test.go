package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWarehouse_GetInterval(t *testing.T) {
	tests := []struct {
		name        string
		warehouse   *Warehouse
		minInterval time.Duration
		assertions  func(t *testing.T, w *Warehouse, interval time.Duration, minInterval time.Duration)
	}{
		{
			name: "no discovery has taken place yet, spec interval > min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 5 * time.Minute},
				},
			},
			minInterval: 2 * time.Minute,
			assertions: func(t *testing.T, warehouse *Warehouse, interval time.Duration, _ time.Duration) {
				require.Equal(t, warehouse.Spec.Interval.Duration, interval)
			},
		},
		{
			name: "no discovery has taken place yet, spec interval < min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 2 * time.Minute},
				},
			},
			minInterval: 5 * time.Minute,
			assertions: func(t *testing.T, warehouse *Warehouse, interval time.Duration, minInterval time.Duration) {
				require.Equal(t, minInterval, interval)
				require.Greater(t, interval, warehouse.Spec.Interval.Duration)
			},
		},
		{
			name: "next discovery is overdue, spec interval > min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 5 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-6 * time.Minute)),
					},
				},
			},
			minInterval: 2 * time.Minute,
			assertions: func(t *testing.T, _ *Warehouse, interval time.Duration, _ time.Duration) {
				require.Equal(t, interval, 100 *time.Millisecond)
			},
		},
		{
			name: "next discovery is overdue, spec interval < min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 2 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-6 * time.Minute)),
					},
				},
			},
			minInterval: 5 * time.Minute,
			assertions: func(t *testing.T, _ *Warehouse, interval time.Duration, _ time.Duration) {
				require.Equal(t, interval, 100 *time.Millisecond)
			},
		},
		{
			name: "next discovery is not overdue, spec interval > min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 5 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-3 * time.Minute)),
					},
				},
			},
			minInterval: 2 * time.Minute,
			assertions: func(t *testing.T, w *Warehouse, interval time.Duration, _ time.Duration) {
				require.NotZero(t, interval)
				require.Less(t, interval, w.Spec.Interval.Duration)
				// Should be around 2 minutes (5 - 3)
				require.InDelta(t, 2*time.Minute, interval, float64(10*time.Second))
			},
		},
		{
			name: "next discovery is not overdue, spec interval < min",
			warehouse: &Warehouse{
				Spec: WarehouseSpec{
					Interval: metav1.Duration{Duration: 2 * time.Minute},
				},
				Status: WarehouseStatus{
					DiscoveredArtifacts: &DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
					},
				},
			},
			minInterval: 5 * time.Minute,
			assertions: func(t *testing.T, w *Warehouse, interval time.Duration, _ time.Duration) {
				require.NotZero(t, interval)
				// Should be around 4 minutes (5 - 1) since effective interval is min (5 minutes)
				require.InDelta(t, 4*time.Minute, interval, float64(10*time.Second))
				require.Greater(t, interval, w.Spec.Interval.Duration)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, tt.warehouse, tt.warehouse.GetInterval(tt.minInterval), tt.minInterval)
		})
	}
}
