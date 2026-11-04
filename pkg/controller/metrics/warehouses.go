package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/metrics"
)

// discoveryDurationBuckets are the buckets for artifact discovery time, a
// tenth of a second to five minutes.
var discoveryDurationBuckets = []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300}

// discoveryIntervalBuckets are the buckets for the time between a Warehouse's
// discoveries, half a minute to an hour.
var discoveryIntervalBuckets = []float64{30, 60, 120, 300, 450, 600, 900, 1800, 3600}

var (
	warehouseMetricsOnce = sync.Once{}
	warehouseMetrics     *WarehouseMetrics
)

// WarehouseMetrics describes Warehouse activity, labeled by Project.
type WarehouseMetrics struct {
	// DiscoveryDuration observes, in seconds, each successful discovery.
	DiscoveryDuration *prometheus.HistogramVec
	// DiscoveryInterval observes, in seconds, the time since a Warehouse's
	// previous successful discovery.
	DiscoveryInterval *prometheus.HistogramVec
	// FreightCreated counts the Freight created by Warehouses.
	FreightCreated *prometheus.CounterVec
}

// NewWarehouseMetrics returns the WarehouseMetrics singleton, registering its
// collectors and a scrape-time collector reading c on first call.
func NewWarehouseMetrics(c client.Client) *WarehouseMetrics {
	warehouseMetricsOnce.Do(func() {
		warehouseMetrics = initWarehouseMetrics(c)
	})
	return warehouseMetrics
}

func initWarehouseMetrics(c client.Client) *WarehouseMetrics {
	histogram := func(name, help string, buckets []float64) *prometheus.HistogramVec {
		return prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:                       metrics.Namespace,
				Name:                            name,
				Help:                            help,
				Buckets:                         buckets,
				NativeHistogramBucketFactor:     metrics.NativeHistogramBucketFactor,
				NativeHistogramMaxBucketNumber:  metrics.NativeHistogramMaxBucketNumber,
				NativeHistogramMinResetDuration: metrics.NativeHistogramMinResetDuration,
			},
			[]string{metrics.ProjectLabel},
		)
	}
	m := &WarehouseMetrics{
		DiscoveryDuration: histogram(
			"warehouse_discovery_duration_seconds",
			"Time a Warehouse's successful artifact discovery took, across all of its subscriptions.",
			discoveryDurationBuckets,
		),
		DiscoveryInterval: histogram(
			"warehouse_discovery_interval_seconds",
			"Time between a Warehouse's previous successful artifact discovery and its latest one.",
			discoveryIntervalBuckets,
		),
		FreightCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "warehouse_freight_created_total",
				Help:      "Number of Freight created by Warehouses.",
			},
			[]string{metrics.ProjectLabel},
		),
	}
	registerLifecycle(
		ctrlmetrics.Registry,
		m.DiscoveryDuration,
		m.DiscoveryInterval,
		m.FreightCreated,
		newWarehouseDiscoveryCollector(c),
	)
	return m
}

// warehouseDiscoveryCollector reports each Project's least recent Warehouse
// discovery time.
type warehouseDiscoveryCollector struct {
	client client.Client
	desc   *prometheus.Desc
}

func newWarehouseDiscoveryCollector(c client.Client) *warehouseDiscoveryCollector {
	return &warehouseDiscoveryCollector{
		client: c,
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(metrics.Namespace, "", "warehouse_oldest_discovery_timestamp_seconds"),
			"Time, as a Unix timestamp, of the least recent successful artifact discovery among "+
				"the Project's Warehouses. Warehouses that have never discovered artifacts are "+
				"not considered.",
			[]string{metrics.ProjectLabel},
			nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (w *warehouseDiscoveryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- w.desc
}

// Collect implements prometheus.Collector.
func (w *warehouseDiscoveryCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	warehouses := &kargoapi.WarehouseList{}
	// Read-only, so skip the deep copy.
	if err := w.client.List(ctx, warehouses, client.UnsafeDisableDeepCopy); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err, "Error when attempting to list warehouse cache during metrics scrape",
		)
		return
	}
	for project, discovered := range oldestDiscoveryByProject(warehouses.Items) {
		ch <- prometheus.MustNewConstMetric(
			w.desc, prometheus.GaugeValue, float64(discovered.Unix()), project,
		)
	}
}

// oldestDiscoveryByProject returns each Project's least recent Warehouse
// discovery time, skipping Warehouses that never discovered.
func oldestDiscoveryByProject(warehouses []kargoapi.Warehouse) map[string]time.Time {
	oldest := make(map[string]time.Time)
	for i := range warehouses {
		wh := &warehouses[i]
		artifacts := wh.Status.DiscoveredArtifacts
		if artifacts == nil || artifacts.DiscoveredAt.IsZero() {
			continue
		}
		discovered := artifacts.DiscoveredAt.Time
		if cur, ok := oldest[wh.Namespace]; !ok || discovered.Before(cur) {
			oldest[wh.Namespace] = discovered
		}
	}
	return oldest
}
