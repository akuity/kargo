package targets

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func jsonBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

func TestCreate(t *testing.T) {
	t.Parallel()
	newTarget := func() *kargoapi.Target {
		return &kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "us-east-1",
				Labels: map[string]string{"region": "us"},
			},
			Spec: kargoapi.TargetSpec{
				Params: map[string]apiextensionsv1.JSON{
					"cluster": {Raw: []byte(`{"replicas":3}`)},
				},
			},
		}
	}
	var authorizations []can.Access
	forbiddenStore := &fakeStore{}
	existingStore := &fakeStore{}
	existingStore.add(testProject, "us-east-1", nil)
	invalidStore := &fakeStore{}

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodPost,
			url:    projectURL(""),
			body:   jsonBody(t, newTarget()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodPost,
			url:       projectURL(""),
			body:      jsonBody(t, newTarget()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "malformed body",
			store:  invalidStore,
			method: http.MethodPost,
			url:    projectURL(""),
			body:   bytes.NewBufferString("{nope"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Zero(t, invalidStore.calls)
			},
		},
		{
			name:   "namespace does not match the Project",
			store:  invalidStore,
			method: http.MethodPost,
			url:    projectURL(""),
			body: jsonBody(t, &kargoapi.Target{
				ObjectMeta: metav1.ObjectMeta{Namespace: "other", Name: "us-east-1"},
			}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), "namespace in body")
			},
		},
		{
			name:   "invalid Target",
			store:  invalidStore,
			method: http.MethodPost,
			url:    projectURL(""),
			body:   jsonBody(t, &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Name: "Not_Valid"}}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, w.Code)
				require.Contains(t, w.Body.String(), "metadata.name")
				require.Zero(t, invalidStore.calls)
			},
		},
		{
			name:   "Target already exists",
			store:  existingStore,
			method: http.MethodPost,
			url:    projectURL(""),
			body:   jsonBody(t, newTarget()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			name:   "store failure",
			store:  &fakeStore{err: errors.New("boom")},
			method: http.MethodPost,
			url:    projectURL(""),
			body:   jsonBody(t, newTarget()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
				require.NotContains(t, w.Body.String(), "boom")
			},
		},
		{
			name:      "creates the Target",
			store:     &fakeStore{},
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodPost,
			url:       projectURL(""),
			body: jsonBody(t, func() *kargoapi.Target {
				// Status and server-managed metadata are ignored.
				target := newTarget()
				target.UID = "client-chosen"
				target.ResourceVersion = "999"
				target.Status.SetStatusForStage("stage", kargoapi.TargetStageStatus{})
				return target
			}()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, w.Code)
				target := &kargoapi.Target{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
				require.Equal(t, testProject, target.Namespace)
				require.Equal(t, "us-east-1", target.Name)
				require.Equal(t, map[string]string{"region": "us"}, target.Labels)
				require.JSONEq(t, `{"replicas":3}`, string(target.Spec.Params["cluster"].Raw))
				require.NotEqual(t, "client-chosen", string(target.UID))
				require.NotEqual(t, "999", target.ResourceVersion)
				require.Empty(t, target.Status.Stages)
				require.Equal(t, []can.Access{can.Create().Target(testProject, "")}, authorizations)
			},
		},
	})
}
