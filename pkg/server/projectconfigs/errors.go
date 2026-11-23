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

// projectConfigsResource is the group and resource ProjectConfigs are
// reported as in errors.
var projectConfigsResource = kargoapi.GroupVersion.WithResource("projectconfigs").GroupResource()

// storeError translates what the store reports into the error a
// Kubernetes-backed handler would have produced for the same situation, so
// that clients see the same status codes whichever store backs the resource.
// A ProjectConfig is named after its Project.
func storeError(err error, project string) error {
	gr := projectConfigsResource
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

// translatedError is an error from the store together with what storeError
// translated it to. It reads as the translation, which is what the client
// is answered with, and unwraps to both, so that code which looks for the
// store's sentinel errors, such as a watch, still finds them.
type translatedError struct {
	translated error
	cause      error
}

func (e *translatedError) Error() string { return e.translated.Error() }

func (e *translatedError) Unwrap() []error { return []error{e.translated, e.cause} }
