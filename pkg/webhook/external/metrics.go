package external

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/akuity/kargo/pkg/metrics"
)

const (
	// receiverTypeLabel is the label under which a webhook receiver's type is
	// recorded, or "unknown" when the request matched no usable receiver.
	receiverTypeLabel = "receiverType"
	// codeLabel is the label under which the response's HTTP status code is
	// recorded.
	codeLabel = "code"
)

var (
	metricsOnce sync.Once
	// requests counts inbound webhook requests; nil until registerMetrics.
	requests *prometheus.CounterVec
	// warehousesSkipped counts Warehouses a webhook could not refresh because a
	// subscription could not be evaluated; nil until registerMetrics.
	warehousesSkipped *prometheus.CounterVec
)

// registerMetrics creates and registers the server's metrics on first call.
func registerMetrics() {
	metricsOnce.Do(func() {
		requests = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "external_webhook_requests_total",
				Help:      "Number of inbound webhook requests, by receiver type and response status code.",
			},
			[]string{metrics.ProjectLabel, receiverTypeLabel, codeLabel},
		)
		warehousesSkipped = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metrics.Namespace,
				Name:      "external_webhook_warehouses_skipped_total",
				Help: "Number of Warehouses a webhook did not refresh because one of their " +
					"subscriptions could not be evaluated, by the Warehouse's Project.",
			},
			[]string{metrics.ProjectLabel},
		)
		ctrlmetrics.Registry.MustRegister(requests, warehousesSkipped)
	})
}

// statusRecorder records the status code written to an http.ResponseWriter.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

// WriteHeader implements http.ResponseWriter.
func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}
