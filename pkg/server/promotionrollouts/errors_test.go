package promotionrollouts

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func TestStoreError(t *testing.T) {
	t.Parallel()
	require.True(t, apierrors.IsNotFound(storeError(database.ErrNotFound, "x")))
	require.True(t, apierrors.IsAlreadyExists(storeError(database.ErrAlreadyExists, "x")))
	require.True(t, apierrors.IsConflict(storeError(database.ErrConflict, "x")))
	var httpErr *libhttp.HTTPError
	require.ErrorAs(t, storeError(database.ErrInvalid, "x"), &httpErr)
	require.Equal(t, 422, httpErr.Code())
	require.ErrorAs(t, storeError(database.ErrNotMirrored, "x"), &httpErr)
	require.Equal(t, 503, httpErr.Code())
	other := errors.New("boom")
	require.Same(t, other, storeError(other, "x"))
}

func TestNoTargetsError(t *testing.T) {
	t.Parallel()
	err := newNoTargetsError("prod")
	require.ErrorIs(t, err, ErrNoTargets)
	var httpErr *libhttp.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusConflict, httpErr.Code())
	// The message is the one the API answers with.
	require.Equal(t, httpErr.Error(), err.Error())
	require.ErrorContains(t, err, `Stage "prod" governs no Targets right now`)
	// Wrapped by a caller, it is still recognized as both.
	wrapped := errors.Join(errors.New("another Stage failed"), err)
	require.ErrorIs(t, wrapped, ErrNoTargets)
	require.ErrorAs(t, wrapped, &httpErr)
}
