package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestNewFreightMetrics(t *testing.T) {
	t.Parallel()
	m := NewFreightMetrics()
	require.Same(t, m, NewFreightMetrics())
	m.VerificationLeadTime.WithLabelValues("project-a").Observe(60)
	count, err := testutil.GatherAndCount(
		ctrlmetrics.Registry,
		"kargo_freight_verification_lead_time_seconds",
	)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
