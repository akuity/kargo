package telemetry

import (
	"context"
	"net/http"
	"testing"

	"k8s.io/client-go/rest"
)

// BenchmarkKubernetesRequest measures a request through a REST config
// WrapRESTConfig wrapped, without TLS, the network or decoding.
//
//	go test -run '^$' -bench KubernetesRequest -benchtime 50000x -count 5 ./pkg/telemetry/
func BenchmarkKubernetesRequest(b *testing.B) {
	const url = "https://kube/apis/kargo.akuity.io/v1alpha1/namespaces/smoke/freights/abc/status"
	for _, m := range benchModes(b) {
		b.Run(m.name, func(b *testing.B) {
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
				req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, nil)
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
