package targets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestUpdate(t *testing.T) {
	t.Parallel()
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "us-east-1", map[string]string{"region": "us"})
		return store
	}
	relabeled := func() *kargoapi.Target {
		return &kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "us-east-1",
				Labels: map[string]string{"region": "eu"},
			},
		}
	}
	var authorizations []authorizationCall
	forbiddenStore := newStore()
	invalidStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodPut,
			url:    projectURL("/us-east-1"),
			body:   jsonBody(t, relabeled()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodPut,
			url:       projectURL("/us-east-1"),
			body:      jsonBody(t, relabeled()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "name does not match the URL",
			store:  invalidStore,
			method: http.MethodPut,
			url:    projectURL("/us-east-1"),
			body:   jsonBody(t, &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Name: "us-west-2"}}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), "name in body")
				require.Zero(t, invalidStore.calls)
			},
		},
		{
			name:   "invalid Target",
			store:  invalidStore,
			method: http.MethodPut,
			url:    projectURL("/us-east-1"),
			body: jsonBody(t, &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "us-east-1",
					Labels: map[string]string{"bad label": "x"},
				},
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "metadata.labels")
			},
		},
		{
			name:   "Target does not exist",
			store:  &fakeStore{},
			method: http.MethodPut,
			url:    projectURL("/us-east-1"),
			body:   jsonBody(t, relabeled()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "stale resource version",
			store:  newStore(),
			method: http.MethodPut,
			url:    projectURL("/us-east-1"),
			body: jsonBody(t, &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{Name: "us-east-1", ResourceVersion: "stale"},
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			name:      "updates the Target, taking its name from the URL",
			store:     newStore(),
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodPut,
			url:       projectURL("/us-east-1"),
			body: jsonBody(t, &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"region": "eu"}},
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				target := &kargoapi.Target{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
				require.Equal(t, "us-east-1", target.Name)
				require.Equal(t, map[string]string{"region": "eu"}, target.Labels)
				require.Equal(t, []authorizationCall{{
					verb: "update",
					gvr:  kargoapi.GroupVersion.WithResource("targets"),
					key:  client.ObjectKey{Namespace: testProject, Name: "us-east-1"},
				}}, authorizations)
			},
		},
	})
}
