package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/akuity/kargo/pkg/metrics"
)

// verificationLeadTimeBuckets are the buckets for Freight verification lead
// time, half a minute to a day.
var verificationLeadTimeBuckets = []float64{30, 60, 120, 300, 600, 1800, 3600, 7200, 21600, 86400}

var (
	freightMetricsOnce = sync.Once{}
	freightMetrics     *FreightMetrics
)

// FreightMetrics describes Freight activity, labeled by Project.
type FreightMetrics struct {
	// VerificationLeadTime observes, in seconds, the time from a Freight's
	// creation until it was verified in a Stage, once per Stage.
	VerificationLeadTime *prometheus.HistogramVec
}

// NewFreightMetrics returns the FreightMetrics singleton, registering its
// collectors with the controller-runtime registry on first call.
func NewFreightMetrics() *FreightMetrics {
	freightMetricsOnce.Do(func() {
		freightMetrics = &FreightMetrics{
			VerificationLeadTime: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: metrics.Namespace,
					Name:      "freight_verification_lead_time_seconds",
					Help: "Time from a Freight's creation until it was verified in a Stage, " +
						"observed once per Stage.",
					Buckets:                         verificationLeadTimeBuckets,
					NativeHistogramBucketFactor:     metrics.NativeHistogramBucketFactor,
					NativeHistogramMaxBucketNumber:  metrics.NativeHistogramMaxBucketNumber,
					NativeHistogramMinResetDuration: metrics.NativeHistogramMinResetDuration,
				},
				[]string{metrics.ProjectLabel},
			),
		}
		ctrlmetrics.Registry.MustRegister(freightMetrics.VerificationLeadTime)
	})
	return freightMetrics
}
