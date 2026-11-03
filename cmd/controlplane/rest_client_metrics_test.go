package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
	enableRESTClientMetrics()
	enableRESTClientMetrics()

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		// A host name, rather than an IP, so the client resolves it.
		Host:  strings.Replace(srv.URL, "127.0.0.1", "localhost", 1),
		QPS:   100,
		Burst: 1,
	})
	require.NoError(t, err)
	_, err = clientset.CoreV1().RESTClient().Get().AbsPath("/healthz").Do(t.Context()).Raw()
	require.NoError(t, err)

	families, err := ctrlmetrics.Registry.Gather()
	require.NoError(t, err)
	samples := map[string]float64{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			samples[family.GetName()] += float64(metric.GetHistogram().GetSampleCount()) +
				metric.GetCounter().GetValue()
		}
	}

	for _, name := range []string{
		"rest_client_request_duration_seconds",
		"rest_client_rate_limiter_duration_seconds",
		"rest_client_request_retries_total",
		"rest_client_request_size_bytes",
		"rest_client_response_size_bytes",
		"rest_client_dns_resolution_duration_seconds",
	} {
		require.Positive(t, samples[name], "expected samples for %s", name)
	}
}
