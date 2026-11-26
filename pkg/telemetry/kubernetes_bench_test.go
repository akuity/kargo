package telemetry

import (
	"context"
	"net/http"
	"testing"

	"k8s.io/client-go/rest"
)

// BenchmarkKubernetesRequest measures requests through a REST config
// WrapRESTConfig wrapped, without TLS, the network or decoding.
//
//	go test -run '^$' -bench KubernetesRequest -benchtime 50000x -count 5 ./pkg/telemetry/
func BenchmarkKubernetesRequest(b *testing.B) {
	requests := []struct {
		name   string
		method string
		url    string
	}{
		{
			name:   "get-stage",
			method: http.MethodGet,
			url:    "https://kube/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/stages/dev",
		},
		{
			name:   "list-analysisruns",
			method: http.MethodGet,
			url:    "https://kube/apis/argoproj.io/v1alpha1/namespaces/smoke/analysisruns?labelSelector=a%3Db",
		},
		{
			name:   "create-promotion",
			method: http.MethodPost,
			url:    "https://kube/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/promotions",
		},
	}
	for _, r := range requests {
		for _, m := range benchModes(b) {
			b.Run(r.name+"/"+m.name, func(b *testing.B) {
				useBenchMode(b, m)
				cfg := &rest.Config{Host: "https://kube", Transport: okRoundTripper{}}
				WrapRESTConfig(cfg)
				rt, err := rest.TransportFor(cfg)
				if err != nil {
					b.Fatal(err)
				}
				ctx := ContextWithAttributes(
					context.Background(),
					ProjectKey.String("smoke"),
					StageKey.String("dev"),
				)
				ctx, span := m.tracer().Start(ctx, "Reconcile Stage")
				defer span.End()
				b.ReportAllocs()
				for b.Loop() {
					req, err := http.NewRequestWithContext(ctx, r.method, r.url, nil)
					if err != nil {
						b.Fatal(err)
					}
					res, err := rt.RoundTrip(req)
					if err != nil {
						b.Fatal(err)
					}
					_ = res.Body.Close()
				}
			})
		}
	}
}
