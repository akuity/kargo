package targets

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDelete(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "us-east-1", nil)
		return store
	}
	var authorizations []authorizationCall
	forbiddenStore := newStore()
	deletedStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodDelete,
			url:    projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodDelete,
			url:       projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "Target does not exist",
			store:  &fakeStore{},
			method: http.MethodDelete,
			url:    projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:      "deletes the Target",
			store:     deletedStore,
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodDelete,
			url:       projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, w.Code)
				_, err := deletedStore.GetTarget(t.Context(), testProject, "us-east-1")
				require.Error(t, err)
				require.Equal(t, []authorizationCall{{
					verb: "delete",
					gvr:  kargoapi.GroupVersion.WithResource("targets"),
					key:  client.ObjectKey{Namespace: testProject, Name: "us-east-1"},
				}}, authorizations)
			},
		},
	})
}
