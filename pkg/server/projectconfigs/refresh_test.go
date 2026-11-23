package projectconfigs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

func TestRefresh(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(kargoapi.ProjectConfigSpec{})
		return store
	}
	forbiddenStore := newStore()
	refreshedStore := newStore()
	var authorizations []can.Access

	runTestCases(t, []testCase{
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodPost,
			url:       configURL("/refresh"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "Project has no ProjectConfig",
			method: http.MethodPost,
			url:    configURL("/refresh"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:      "returns the refresh token",
			store:     refreshedStore,
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodPost,
			url:       configURL("/refresh"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []can.Access{can.Get().ProjectConfig(testProject)}, authorizations)
				var resp wire.RefreshResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.NotEmpty(t, resp.Token)
				stored := refreshedStore.configs[testProject].RefreshToken
				require.NotNil(t, stored)
				require.Equal(t, *stored, resp.Token)
			},
		},
	})
}
