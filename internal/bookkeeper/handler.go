package bookkeeper

import (
	"net/http"
)

// handler is an implementation of the http.Handler interface that can handle
// HTTP-based bookkeeping requests by delegating to a transport-agnostic Service
// interface.
type handler struct {
	service Service
}

// NewHandler returns an implementation of the http.Handler interface that can
// handle HTTP-based bookkeeping requests by delegating to a transport-agnostic
// Service interface.
func NewHandler(service Service) http.Handler {
	return &handler{
		service: service,
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	// TODO: Get some payload and send it to the service
	if err := h.service.Handle(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`)) // nolint: errcheck
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`)) // nolint: errcheck
}
