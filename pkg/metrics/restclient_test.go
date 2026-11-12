package metrics

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestEnableRESTClientMetrics(t *testing.T) {
	// Throttle the first request so the client retries it.
	var throttled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if throttled.CompareAndSwap(false, true) {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	// Enabling twice must not panic on duplicate registration.
	EnableRESTClientMetrics()
	EnableRESTClientMetrics()

	// A host name, so the client resolves it.
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host: "http://" + net.JoinHostPort("localhost", port),
	})
	require.NoError(t, err)

	// The registry is global, so compare against what was recorded before.
	before := gatherRESTClientSamples(t)
	_, err = clientset.CoreV1().RESTClient().Get().AbsPath("/healthz").Do(t.Context()).Raw()
	require.NoError(t, err)

	after := gatherRESTClientSamples(t)

	for _, name := range []string{
		"rest_client_request_duration_seconds",
		"rest_client_rate_limiter_duration_seconds",
		"rest_client_request_retries_total",
		"rest_client_request_size_bytes",
		"rest_client_response_size_bytes",
		"rest_client_dns_resolution_duration_seconds",
	} {
		require.Greater(t, after[name], before[name], "expected new samples for %s", name)
	}
}

// gatherRESTClientSamples returns each metric family's sample count.
func gatherRESTClientSamples(t *testing.T) map[string]float64 {
	t.Helper()
	families, err := ctrlmetrics.Registry.Gather()
	require.NoError(t, err)
	samples := map[string]float64{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			samples[family.GetName()] += float64(metric.GetHistogram().GetSampleCount()) +
				metric.GetCounter().GetValue()
		}
	}
	return samples
}
