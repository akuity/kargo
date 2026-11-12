package server

import "net/http"

const (
	livenessPath  = "/healthz"
	readinessPath = "/readyz"
)

// HealthChecker reports on the health of something the server depends on, in
// the terms used by Kubernetes probes.
type HealthChecker interface {
	// Ready returns an error if the server should not receive traffic.
	Ready() error
	// Live returns an error if the server should be restarted.
	Live() error
}

// handleHealth registers the endpoints for Kubernetes probes on mux.
func (s *server) handleHealth(mux *http.ServeMux) {
	var live, ready func() error
	if s.health != nil {
		live, ready = s.health.Live, s.health.Ready
	}
	mux.Handle(livenessPath, newHealthHandler(live))
	mux.Handle(readinessPath, newHealthHandler(ready))
}

// newHealthHandler returns a handler for Kubernetes probes. It responds 200 if
// check returns nil, or 503 with the error otherwise. A nil check always
// passes.
func newHealthHandler(check func() error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body := "ok"
		status := http.StatusOK
		if check != nil {
			if err := check(); err != nil {
				body = err.Error()
				status = http.StatusServiceUnavailable
			}
		}
		w.WriteHeader(status)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(body))
		}
	})
}
