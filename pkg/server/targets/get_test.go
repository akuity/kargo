package targets

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
		store.add(testProject, "us-east-1", map[string]string{"region": "us"})
		return store
	}
	var authorizations []can.Access
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
				require.Equal(t, []can.Access{can.Get().Target(testProject, "us-east-1")}, authorizations)
				require.Empty(t, target.Status.Stages)
			},
		},
		{
			name: "returns the Target with its status, and only its own",
			store: func() *fakeStore {
				store := newStore()
				store.add(testProject, "us-west-2", nil)
				store.addFreight("us-east-1", "fleet", "f1")
				store.addFreight("us-west-2", "fleet", "f0")
				return store
			}(),
			method: http.MethodGet,
			url:    projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				target := &kargoapi.Target{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
				require.Len(t, target.Status.Stages, 1)
				require.Equal(t, collectionID("f1"), target.Status.Stages["fleet"].CurrentFreight.ID)
			},
		},
		{
			name: "a failure to read its status fails the get",
			store: func() *fakeStore {
				store := newStore()
				store.freightErr = errors.New("offline")
				return store
			}(),
			method: http.MethodGet,
			url:    projectURL("/us-east-1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	})
}
