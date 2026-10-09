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
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestList(t *testing.T) {
	t.Parallel()

	newStage := func(name string, selectors ...metav1.LabelSelector) *kargoapi.Stage {
		stage := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: name},
		}
		if selectors != nil {
			stage.Spec.Targets = &kargoapi.StageTargets{Selectors: selectors}
		}
		return stage
	}

	// Three Targets, added out of name order, plus one in another Project.
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "us-west-2", map[string]string{"region": "us"})
		store.add(testProject, "eu-central-1", map[string]string{"region": "eu", "tier": "canary"})
		store.add(testProject, "us-east-1", map[string]string{"region": "us", "tier": "canary"})
		store.add("other-project", "ap-south-1", map[string]string{"region": "ap"})
		return store
	}

	names := func(t *testing.T, w *httptest.ResponseRecorder) []string {
		require.Equal(t, http.StatusOK, w.Code)
		list := &kargoapi.TargetList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		require.NotNil(t, list.Items)
		out := make([]string, 0, len(list.Items))
		for _, target := range list.Items {
			out = append(out, target.Name)
		}
		return out
	}

	var authorizations []can.Access
	forbiddenStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodGet,
			url:    projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "no Targets exist",
			store:  &fakeStore{},
			method: http.MethodGet,
			url:    projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, names(t, w))
			},
		},
		{
			name:      "lists the Project's Targets sorted by name",
			store:     newStore(),
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodGet,
			url:       projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"eu-central-1", "us-east-1", "us-west-2"}, names(t, w))
				require.Equal(t, []can.Access{can.List().Target(testProject, "")}, authorizations)
			},
		},
		{
			name:   "filters by label selector",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?labelSelector=region%3Dus"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"us-east-1", "us-west-2"}, names(t, w))
			},
		},
		{
			name:   "rejects a malformed label selector",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?labelSelector=region%3D%3D%3D"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "Stage filter names a Stage that does not exist",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=missing"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:    "Stage filter with a classic Stage governs no Targets",
			store:   newStore(),
			objects: []client.Object{newStage("classic")},
			method:  http.MethodGet,
			url:     projectURL("?stage=classic"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, names(t, w))
			},
		},
		{
			name:  "Stage filter unions the Stage's selectors",
			store: newStore(),
			objects: []client.Object{newStage(
				"fleet",
				metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}},
				metav1.LabelSelector{MatchLabels: map[string]string{"region": "eu"}},
			)},
			method: http.MethodGet,
			url:    projectURL("?stage=fleet"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"eu-central-1", "us-east-1", "us-west-2"}, names(t, w))
			},
		},
		{
			name:  "Stage filter combines with a label selector",
			store: newStore(),
			objects: []client.Object{newStage(
				"fleet",
				metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}},
			)},
			method: http.MethodGet,
			url:    projectURL("?stage=fleet&labelSelector=tier%3Dcanary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"us-east-1"}, names(t, w))
			},
		},
	})
}
