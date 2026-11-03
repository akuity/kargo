package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestEnableRESTClientMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	// Enabling twice must not panic on duplicate registration.
	enableRESTClientMetrics()
	enableRESTClientMetrics()

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host:  srv.URL,
		QPS:   1,
		Burst: 1,
	})
	require.NoError(t, err)
	_, err = clientset.CoreV1().RESTClient().Get().AbsPath("/healthz").Do(t.Context()).Raw()
	require.NoError(t, err)

	families, err := ctrlmetrics.Registry.Gather()
	require.NoError(t, err)
	sampleCounts := map[string]uint64{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			sampleCounts[family.GetName()] += metric.GetHistogram().GetSampleCount()
		}
	}

	for _, name := range []string{
		"rest_client_request_duration_seconds",
		"rest_client_rate_limiter_duration_seconds",
		"rest_client_request_size_bytes",
		"rest_client_response_size_bytes",
	} {
		require.Positive(t, sampleCounts[name], "expected samples for %s", name)
	}
}
