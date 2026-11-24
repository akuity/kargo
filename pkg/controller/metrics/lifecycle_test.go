package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func Test_registerLifecycle(t *testing.T) {
	// Not parallel: reads and sets a package-level switch.
	prev := lifecycleMetricsDisabled.Load()
	t.Cleanup(func() { lifecycleMetricsDisabled.Store(prev) })
	newCounter := func() prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{Name: "test_total"})
	}
	gathered := func(t *testing.T, r *prometheus.Registry) int {
		t.Helper()
		families, err := r.Gather()
		require.NoError(t, err)
		return len(families)
	}

	lifecycleMetricsDisabled.Store(false)
	r := prometheus.NewRegistry()
	registerLifecycle(r, newCounter())
	require.Equal(t, 1, gathered(t, r))

	// Once disabled, nothing is registered.
	DisableLifecycleMetrics()
	r = prometheus.NewRegistry()
	registerLifecycle(r, newCounter())
	require.Zero(t, gathered(t, r))
}
