package projectconfigs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestDelete(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(kargoapi.ProjectConfigSpec{})
		return store
	}
	forbiddenStore := newStore()
	deletedStore := newStore()
	leftovers := leftoverStore()
	var authorizations []can.Access

	runTestCases(t, []testCase{
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodDelete,
			url:       configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "ProjectConfig does not exist",
			method: http.MethodDelete,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "a leftover of a deleted Project of the same name is left alone",
			store:  leftovers,
			method: http.MethodDelete,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
				require.Contains(t, leftovers.configs, testProject)
			},
		},
		{
			name:      "deletes the ProjectConfig",
			store:     deletedStore,
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodDelete,
			url:       configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, w.Code)
				require.Empty(t, deletedStore.configs)
				require.Equal(t, []can.Access{can.Delete().ProjectConfig(testProject)}, authorizations)
			},
		},
	})
}
