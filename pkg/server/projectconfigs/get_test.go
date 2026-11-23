package projectconfigs

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestGet(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(kargoapi.ProjectConfigSpec{
			FreightLinks: []kargoapi.DeepLink{{Title: "t", URL: "https://example.com"}},
		})
		return store
	}
	forbiddenStore := newStore()
	var authorizations []can.Access

	runTestCases(t, []testCase{
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "ProjectConfig does not exist",
			method: http.MethodGet,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "a leftover of a deleted Project of the same name is not found",
			store:  leftoverStore(),
			method: http.MethodGet,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "store fails",
			store:  &fakeStore{err: errors.New("boom")},
			method: http.MethodGet,
			url:    configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name:   "watching is not implemented yet",
			store:  newStore(),
			method: http.MethodGet,
			url:    configURL("?watch=true"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "returns the ProjectConfig with its type",
			store:     newStore(),
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodGet,
			url:       configURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []can.Access{can.Get().ProjectConfig(testProject)}, authorizations)
				var raw map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
				require.Equal(t, "kargo.akuity.io/v1alpha1", raw["apiVersion"])
				require.Equal(t, "ProjectConfig", raw["kind"])
				cfg := &kargoapi.ProjectConfig{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), cfg))
				require.Equal(t, testProject, cfg.Name)
				require.Equal(t, testProject, cfg.Namespace)
				require.Len(t, cfg.Spec.FreightLinks, 1)
			},
		},
	})
}
