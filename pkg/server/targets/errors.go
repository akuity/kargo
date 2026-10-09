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
	gr := kargoapi.GroupVersion.WithResource("targets").GroupResource()
	switch {
	case errors.Is(err, database.ErrNotFound):
		return apierrors.NewNotFound(gr, name)
	case errors.Is(err, database.ErrAlreadyExists):
		return apierrors.NewAlreadyExists(gr, name)
	case errors.Is(err, database.ErrConflict):
		return apierrors.NewConflict(gr, name, err)
	case errors.Is(err, database.ErrInvalid):
		return libhttp.Error(err, http.StatusUnprocessableEntity)
	}
	return err
}
