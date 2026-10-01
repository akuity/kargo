package targets

import (
	"errors"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

// storeError translates what the store reports into the error a
// Kubernetes-backed handler would have produced for the same situation, so
// that clients see the same status codes whichever store backs the resource.
func storeError(err error, name string) error {
	gr := kargoapi.GroupVersion.WithResource(resource).GroupResource()
	switch {
	case errors.Is(err, database.ErrNotFound):
		return httpError(apierrors.NewNotFound(gr, name))
	case errors.Is(err, database.ErrAlreadyExists):
		return httpError(apierrors.NewAlreadyExists(gr, name))
	case errors.Is(err, database.ErrConflict):
		return httpError(apierrors.NewConflict(gr, name, err))
	case errors.Is(err, database.ErrInvalid):
		return libhttp.Error(err, http.StatusUnprocessableEntity)
	case errors.Is(err, database.ErrProjectNotMirrored):
		// The Project exists in Kubernetes, since the request got this far,
		// but has not reached the database yet. That resolves itself shortly.
		return libhttp.Error(err, http.StatusServiceUnavailable)
	}
	return err
}

// httpError gives a Kubernetes status error the HTTP status it carries, so
// that it is reported with that status rather than as an unexpected failure.
// Any other error is returned as is.
func httpError(err error) error {
	var statusErr *apierrors.StatusError
	if errors.As(err, &statusErr) {
		return libhttp.Error(err, int(statusErr.Status().Code))
	}
	return err
}
