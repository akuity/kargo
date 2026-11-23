package projectconfigs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestPut(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(kargoapi.ProjectConfigSpec{})
		return store
	}
	body := func(s string) *strings.Reader { return strings.NewReader(s) }
	const validBody = `{
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"kind": "ProjectConfig",
		"metadata": {"name": "` + testProject + `", "namespace": "` + testProject + `"},
		"spec": {"freightLinks": [{"title": "t", "url": "https://example.com"}]}
	}`

	forbiddenStore := newStore()
	createForbiddenStore := &fakeStore{}
	updateForbiddenStore := newStore()
	createdStore := &fakeStore{}
	updatedStore := newStore()
	racedStore := &fakeStore{raceOnCreate: true}
	refreshStore := newStore()
	staleRVStore := &fakeStore{}
	staleUIDStore := &fakeStore{}
	var createAuthz, updateAuthz, raceAuthz []can.Access

	runTestCases(t, []testCase{
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodPut,
			url:       configURL(""),
			body:      body(validBody),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:      "may not create",
			store:     createForbiddenStore,
			authorize: forbid(can.Create()),
			method:    http.MethodPut,
			url:       configURL(""),
			body:      body(validBody),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Nil(t, createForbiddenStore.written)
				require.Empty(t, createForbiddenStore.configs)
			},
		},
		{
			name:      "may not update",
			store:     updateForbiddenStore,
			authorize: forbid(can.Update()),
			method:    http.MethodPut,
			url:       configURL(""),
			body:      body(validBody),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Nil(t, updateForbiddenStore.written)
			},
		},
		{
			name:   "malformed body",
			method: http.MethodPut,
			url:    configURL(""),
			body:   body(`{`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:      "creates when absent",
			store:     createdStore,
			authorize: recordAuthorizations(&createAuthz),
			method:    http.MethodPut,
			url:       configURL(""),
			body:      body(validBody),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				require.Equal(t, []can.Access{
					can.Get().ProjectConfig(testProject),
					can.Create().ProjectConfig(testProject),
				}, createAuthz)
				cfg := decodeConfig(t, w)
				require.Equal(t, "ProjectConfig", cfg.Kind)
				require.Equal(t, kargoapi.GroupVersion.String(), cfg.APIVersion)
				require.Len(t, cfg.Spec.FreightLinks, 1)
				require.Contains(t, createdStore.configs, testProject)
			},
		},
		{
			name:   "an absent name is the Project's",
			method: http.MethodPut,
			url:    configURL(""),
			body:   body(`{"spec": {}}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				require.Equal(t, testProject, decodeConfig(t, w).Name)
			},
		},
		{
			name:      "updates when present",
			store:     updatedStore,
			authorize: recordAuthorizations(&updateAuthz),
			method:    http.MethodPut,
			url:       configURL(""),
			body:      body(validBody),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []can.Access{
					can.Get().ProjectConfig(testProject),
					can.Update().ProjectConfig(testProject),
				}, updateAuthz)
				cfg := decodeConfig(t, w)
				require.Equal(t, "ProjectConfig", cfg.Kind)
				require.Equal(t, int64(2), cfg.Generation)
				require.Len(t, cfg.Spec.FreightLinks, 1)
			},
		},
		{
			name:      "a create that loses a race is retried as an update",
			store:     racedStore,
			authorize: recordAuthorizations(&raceAuthz),
			method:    http.MethodPut,
			url:       configURL(""),
			body:      body(validBody),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []can.Access{
					can.Get().ProjectConfig(testProject),
					can.Create().ProjectConfig(testProject),
					can.Update().ProjectConfig(testProject),
				}, raceAuthz)
				require.Equal(t, int64(2), decodeConfig(t, w).Generation)
			},
		},
		{
			name:   "name mismatch on create",
			method: http.MethodPut,
			url:    configURL(""),
			body:   body(`{"metadata": {"name": "other"}, "spec": {}}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "metadata.name")
			},
		},
		{
			name:   "name mismatch on update",
			store:  newStore(),
			method: http.MethodPut,
			url:    configURL(""),
			body:   body(`{"metadata": {"name": "other"}, "spec": {}}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "metadata.name")
			},
		},
		{
			name:   "decode and validation errors are reported together",
			method: http.MethodPut,
			url:    configURL(""),
			body: body(`{
				"metadata": {"name": "other"},
				"spec": {"freightLinks": [{"title": "t"}]}
			}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "spec.freightLinks[0].url")
				require.Contains(t, w.Body.String(), "metadata.name")
			},
		},
		{
			name:   "stale resource version",
			store:  newStore(),
			method: http.MethodPut,
			url:    configURL(""),
			body: body(`{
				"metadata": {"name": "` + testProject + `", "resourceVersion": "999"},
				"spec": {}
			}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			name:   "a resource version does not re-create a deleted ProjectConfig",
			store:  staleRVStore,
			method: http.MethodPut,
			url:    configURL(""),
			body: body(`{
				"metadata": {"name": "` + testProject + `", "resourceVersion": "7"},
				"spec": {}
			}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
				require.Contains(t, w.Body.String(), "no longer exists")
				require.Empty(t, staleRVStore.configs)
			},
		},
		{
			name:   "a uid does not re-create a deleted ProjectConfig",
			store:  staleUIDStore,
			method: http.MethodPut,
			url:    configURL(""),
			body: body(`{
				"metadata": {"name": "` + testProject + `", "uid": "gone"},
				"spec": {}
			}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
				require.Contains(t, w.Body.String(), "no longer exists")
				require.Empty(t, staleUIDStore.configs)
			},
		},
		{
			name:   "the refresh annotation is not stored",
			store:  refreshStore,
			method: http.MethodPut,
			url:    configURL(""),
			body: body(`{
				"metadata": {
					"name": "` + testProject + `",
					"annotations": {"kargo.akuity.io/refresh": "token", "keep": "me"}
				},
				"spec": {}
			}`),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, map[string]string{"keep": "me"}, refreshStore.written.Annotations)
			},
		},
	})
}

func decodeConfig(t *testing.T, w *httptest.ResponseRecorder) *Resource {
	t.Helper()
	cfg := &Resource{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), cfg))
	return cfg
}
