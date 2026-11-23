package projectconfigs

import (
	"errors"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// kind is the Kind ProjectConfigs are served and reported as.
const kind = "ProjectConfig"

// errWatchNotImplemented answers a request to watch a ProjectConfig, which
// the database-backed store cannot serve yet.
var errWatchNotImplemented = libhttp.ErrorStr(
	"watching a ProjectConfig is not implemented",
	http.StatusNotImplemented,
)

// storeError translates what the store reports into the error a
// Kubernetes-backed handler would have produced for the same situation, so
// that clients see the same status codes whichever store backs the resource.
// A ProjectConfig is named after its Project.
func storeError(err error, project string) error {
	gr := kargoapi.GroupVersion.WithResource("projectconfigs").GroupResource()
	switch {
	case errors.Is(err, database.ErrNotFound):
		return apierrors.NewNotFound(gr, project)
	case errors.Is(err, database.ErrAlreadyExists):
		return apierrors.NewAlreadyExists(gr, project)
	case errors.Is(err, database.ErrConflict):
		return apierrors.NewConflict(gr, project, err)
	case errors.Is(err, database.ErrInvalid):
		return libhttp.Error(err, http.StatusUnprocessableEntity)
	}
	return err
}

// invalid reports field errors in a ProjectConfig, as the Kubernetes API
// server reports a resource its schema or admission webhooks refuse.
func invalid(project string, errs field.ErrorList) error {
	return apierrors.NewInvalid(
		schema.GroupKind{Group: kargoapi.GroupVersion.Group, Kind: kind},
		project,
		errs,
	)
}
