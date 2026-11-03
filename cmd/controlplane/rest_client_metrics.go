package main

import (
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// enableRESTClientMetrics enables the client-go REST client metrics that
// controller-runtime leaves off by default. Out of the box, it registers only
// rest_client_requests_total, which leaves no way to see request latency, time
// spent waiting on the client-side rate limiter (i.e. KUBE_API_QPS/BURST
// throttling), retries, or request and response sizes.
//
// The collectors are registered with the controller-runtime registry, which
// every manager's metrics endpoint serves, so they are exposed wherever
// METRICS_BIND_ADDRESS is set. Calling this more than once is safe.
func enableRESTClientMetrics() {
	ctrlmetrics.RegisterRESTClientMetrics(
		ctrlmetrics.MetricRequestLatency,
		ctrlmetrics.MetricDNSResolutionLatency,
		ctrlmetrics.MetricRequestSize,
		ctrlmetrics.MetricResponseSize,
		ctrlmetrics.MetricRateLimiterLatency,
		ctrlmetrics.MetricRequestRetry,
	)
}
