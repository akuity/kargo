package server

import "net/http"

// newHealthHandler returns a trivial liveness handler used by Kubernetes
// startup, liveness, and readiness probes. It replaces the ConnectRPC-based
// gRPC health check, whose probe binary (grpc_health_probe) carries a
// high-severity CVE.
func newHealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("ok"))
		}
	})
}
