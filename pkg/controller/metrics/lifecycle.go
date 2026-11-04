package metrics

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// lifecycleMetricsDisabled is set by DisableLifecycleMetrics.
var lifecycleMetricsDisabled atomic.Bool

// DisableLifecycleMetrics keeps the lifecycle metrics' collectors from being
// registered; observing them stays safe. Call it before any controller is set
// up.
func DisableLifecycleMetrics() {
	lifecycleMetricsDisabled.Store(true)
}

// registerLifecycle registers cs with r unless lifecycle metrics are disabled.
func registerLifecycle(r prometheus.Registerer, cs ...prometheus.Collector) {
	if lifecycleMetricsDisabled.Load() {
		return
	}
	r.MustRegister(cs...)
}
