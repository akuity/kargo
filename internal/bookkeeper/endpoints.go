package bookkeeper

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/akuityio/k8sta/internal/common/config"
	"github.com/akuityio/k8sta/internal/common/version"
)

// SetupEndpoints registers HTTP request handlers for all Bookkeeper
// functionality. All handlers will delegate processing to the provided
// transport-agnostic Service.
func SetupEndpoints(router *mux.Router, service Service, cfg config.Config) {
	e := &endpoints{
		service: service,
		logger:  log.New(),
	}
	e.logger.SetLevel(cfg.LogLevel)
	router.HandleFunc("/v1alpha1/render", e.render).Methods(http.MethodPost)
	router.HandleFunc("/version", e.version).Methods(http.MethodGet)
}

// endpoints handles HTTP-based bookkeeping requests by delegating to a
// transport-agnostic Service.
type endpoints struct {
	service Service
	logger  *log.Logger
}

func (e *endpoints) render(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var logger = e.logger.WithFields(log.Fields{})

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		// We're going to assume this is because the request body is missing and
		// treat it as a bad request.
		logger.Infof("Error reading request body: %s", err)
		e.handleError(
			&ErrBadRequest{
				Reason: "Bookkeeper server was unable to read the request body",
			},
			w,
			logger,
		)
		return
	}

	req := RenderRequest{}
	if err = json.Unmarshal(bodyBytes, &req); err != nil {
		// The request body must be malformed.
		logger.Infof("Error unmarshaling request body: %s", err)
		e.handleError(
			&ErrBadRequest{
				Reason: "Bookkeeper server was unable to unmarshal the request body",
			},
			w,
			logger,
		)
		return
	}

	// TODO: We should apply some kind of request body validation

	// Now that we have details from the request body, we can attach some more
	// context to the logger.
	logger = logger.WithFields(log.Fields{
		"repo":         req.RepoURL,
		"targetBranch": req.TargetBranch,
	})

	res, err := e.service.RenderConfig(r.Context(), req)
	if err != nil {
		e.handleError(
			errors.Wrap(err, "error handling request"),
			w,
			logger,
		)
		return
	}

	e.writeResponse(w, http.StatusOK, res, logger)
}

func (e *endpoints) version(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	e.writeResponse(
		w,
		http.StatusOK,
		version.GetVersion(),
		e.logger.WithFields(log.Fields{}),
	)
}

func (e *endpoints) handleError(
	err error,
	w http.ResponseWriter,
	logger *log.Entry,
) {
	switch typedErr := errors.Cause(err).(type) {
	case *ErrBadRequest:
		e.writeResponse(w, http.StatusBadRequest, typedErr, logger)
	case *ErrNotFound:
		e.writeResponse(w, http.StatusNotFound, typedErr, logger)
	case *ErrConflict:
		e.writeResponse(w, http.StatusConflict, typedErr, logger)
	case *ErrNotSupported:
		e.writeResponse(w, http.StatusNotImplemented, typedErr, logger)
	case *ErrInternalServer:
		e.writeResponse(w, http.StatusInternalServerError, typedErr, logger)
	default:
		if logger != nil {
			logger.Error(err)
		}
		e.writeResponse(
			w,
			http.StatusInternalServerError,
			&ErrInternalServer{},
			logger,
		)
	}
}

func (e *endpoints) writeResponse(
	w http.ResponseWriter,
	statusCode int,
	response any,
	logger *log.Entry,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	responseBody, err := json.Marshal(response)
	if err != nil && logger != nil {
		logger.Errorf("error marshaling response body: %s", err)
	}
	if _, err := w.Write(responseBody); err != nil && logger != nil {
		logger.Errorf("error writing response body: %s", err)
	}
}
