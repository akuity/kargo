package server

import "net/http"

// newHealthHandler returns a trivial liveness handler used by Kubernetes
// startup, liveness, and readiness probes. It replaces the ConnectRPC-based
// gRPC health check that was removed along with the rest of the deprecated
// ConnectRPC API.
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
