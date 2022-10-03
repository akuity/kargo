package bookkeeper

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/akuityio/k8sta/internal/common/config"
	log "github.com/sirupsen/logrus"
)

// handler is an implementation of the http.Handler interface that can handle
// HTTP-based bookkeeping requests by delegating to a transport-agnostic Service
// interface.
type handler struct {
	service Service
	logger  *log.Logger
}

// NewHandler returns an implementation of the http.Handler interface that can
// handle HTTP-based bookkeeping requests by delegating to a transport-agnostic
// Service interface.
func NewHandler(config config.Config, service Service) http.Handler {
	h := &handler{
		service: service,
		logger:  log.New(),
	}
	h.logger.SetLevel(config.LogLevel)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	var logger = h.logger.WithFields(log.Fields{})

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		// We're going to assume this is because the request body is missing and
		// treat it as a bad request.
		logger.Infof("Error reading request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		// nolint: errcheck
		w.Write([]byte(`{"status": "error reading request body"}`))
		return
	}

	req := Request{}
	if err = json.Unmarshal(bodyBytes, &req); err != nil {
		// The request body must be malformed.
		logger.Infof("Error unmarshaling request body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		// nolint: errcheck
		w.Write([]byte(`{"status": "error unmarshaling request body"}`))
		return
	}

	// TODO: We should apply some kind of request body validation

	// Now that we have details from the request body, we can attach some more
	// context to the logger.
	logger = logger.WithFields(log.Fields{
		"repo":         req.RepoURL,
		"path":         req.Path,
		"targetBranch": req.TargetBranch,
	})

	res, err := h.service.Handle(r.Context(), req)
	if err != nil {
		logger.Errorf("Error handling request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`)) // nolint: errcheck
		return
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		logger.Errorf("Error marshaling response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`)) // nolint: errcheck
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes) // nolint: errcheck
}
