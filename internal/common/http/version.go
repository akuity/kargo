package http

import (
	"encoding/json"
	"net/http"

	"github.com/akuityio/k8sta/internal/common/version"
)

// Version responds to an HTTP/S request with version information.
func Version(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	version := version.GetVersion()
	w.Header().Set("Content-Type", "application/json")
	resBytes, err := json.Marshal(version)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`)) // nolint: errcheck
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resBytes) // nolint: errcheck
}
