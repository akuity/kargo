package targets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGet(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "us-east-1", map[string]string{"region": "us"})
		return store
	}
	var authorizations []authorizationCall
	forbiddenStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodGet,
			url:    projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "Target does not exist",
			store:  &fakeStore{},
			method: http.MethodGet,
			url:    projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:      "returns the Target",
			store:     newStore(),
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodGet,
			url:       projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				target := &kargoapi.Target{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
				require.Equal(t, "us-east-1", target.Name)
				require.Equal(t, testProject, target.Namespace)
				require.Equal(t, map[string]string{"region": "us"}, target.Labels)
				require.NotEmpty(t, target.UID)
				require.NotEmpty(t, target.ResourceVersion)
				require.Equal(t, []authorizationCall{{
					verb: "get",
					gvr:  kargoapi.GroupVersion.WithResource("targets"),
					key:  client.ObjectKey{Namespace: testProject, Name: "us-east-1"},
				}}, authorizations)
			},
		},
	})
}
