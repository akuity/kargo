package metrics

import (
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// EnableRESTClientMetrics enables the client-go REST client metrics that
// controller-runtime leaves off by default. Safe to call more than once.
func EnableRESTClientMetrics() {
	ctrlmetrics.RegisterRESTClientMetrics(
		ctrlmetrics.MetricRequestLatency,
		ctrlmetrics.MetricDNSResolutionLatency,
		ctrlmetrics.MetricRequestSize,
		ctrlmetrics.MetricResponseSize,
		ctrlmetrics.MetricRateLimiterLatency,
		ctrlmetrics.MetricRequestRetry,
	)
}
