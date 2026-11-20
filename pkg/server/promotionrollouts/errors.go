package promotionrollouts

import (
	"errors"
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

var (
	errDatabaseNotConfigured = libhttp.ErrorStr(
		"database is not configured",
		http.StatusNotImplemented,
	)

	errCreateDisabled = libhttp.ErrorStr(
		"creating PromotionRollouts is not supported by this server",
		http.StatusNotImplemented,
	)
)

// ErrNoTargets is what a Creator reports, as part of the error it returns,
// when a rollout would promote to no Targets because the Stage governs none
// at this time. It is exported so that a caller holding a Creator can tell
// this refusal from any other with errors.Is, and wait for the Stage to
// govern Targets rather than treat the promotion as failed.
var ErrNoTargets = errors.New("the Stage governs no Targets")

// noTargetsError is the refusal of a rollout that would promote to no
// Targets. It is both an ErrNoTargets and an error carrying an HTTP status,
// so that a caller holding a Creator and a client of the API can each
// recognize it.
type noTargetsError struct {
	// err carries the message and the status the API answers with.
	err error
}

// newNoTargetsError returns the refusal of a rollout for the named Stage,
// which governs no Targets at this time.
//
// The status is a 409 and not a 400 because nothing is wrong with the
// request: it is at odds with the state of the Stage, and the same request
// succeeds once the Stage's selectors match a Target.
func newNoTargetsError(stage string) error {
	return &noTargetsError{err: libhttp.ErrorStr(
		fmt.Sprintf(
			"Stage %q governs no Targets right now, so there is nothing to promote to; "+
				"promote to it once its selectors match at least one Target",
			stage,
		),
		http.StatusConflict,
	)}
}

func (e *noTargetsError) Error() string {
	return e.err.Error()
}

// Unwrap makes the error an ErrNoTargets to errors.Is and an error with an
// HTTP status to errors.As.
func (e *noTargetsError) Unwrap() []error {
	return []error{ErrNoTargets, e.err}
}

// storeError translates what the store reports into the error a
// Kubernetes-backed handler would have produced for the same situation, so
// that clients see the same status codes whichever store backs the resource.
func storeError(err error, name string) error {
	gr := kargoapi.GroupVersion.WithResource("promotionrollouts").GroupResource()
	switch {
	case errors.Is(err, database.ErrNotFound):
		return apierrors.NewNotFound(gr, name)
	case errors.Is(err, database.ErrAlreadyExists):
		return apierrors.NewAlreadyExists(gr, name)
	case errors.Is(err, database.ErrConflict):
		return apierrors.NewConflict(gr, name, err)
	case errors.Is(err, database.ErrInvalid):
		return libhttp.Error(err, http.StatusUnprocessableEntity)
	case errors.Is(err, database.ErrNotMirrored):
		// The Stage exists, or the caller would not have got this far, but
		// the database has yet to learn of it. That passes on its own.
		return libhttp.Error(err, http.StatusServiceUnavailable)
	}
	return err
}
