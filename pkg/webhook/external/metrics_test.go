package external

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func Test_registerMetrics(t *testing.T) {
	registerMetrics()
	registered := requests
	registerMetrics()
	require.Same(t, registered, requests)
	requests.WithLabelValues("fake-project", github, "200").Inc()
	warehousesSkipped.WithLabelValues("fake-project").Inc()
	count, err := testutil.GatherAndCount(
		ctrlmetrics.Registry,
		"kargo_external_webhook_requests_total",
		"kargo_external_webhook_warehouses_skipped_total",
	)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
