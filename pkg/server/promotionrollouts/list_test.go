package promotionrollouts

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestList(t *testing.T) {
	t.Parallel()

	// Three rollouts across two Stages, plus one in another Project.
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhaseSucceeded, "us-east", "us-west")
		store.add(testProject, "staging.01A.abc", "staging", kargoapi.PromotionRolloutPhasePending, "us-east")
		store.add(testProject, "prod.01B.abc", "prod", kargoapi.PromotionRolloutPhaseRunning, "us-east", "us-west")
		store.add("other-project", "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhasePending)
		return store
	}

	decode := func(t *testing.T, w *httptest.ResponseRecorder) []kargoapi.PromotionRollout {
		require.Equal(t, http.StatusOK, w.Code)
		list := &kargoapi.PromotionRolloutList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		require.NotNil(t, list.Items)
		return list.Items
	}
	names := func(rollouts []kargoapi.PromotionRollout) []string {
		out := make([]string, 0, len(rollouts))
		for _, rollout := range rollouts {
			out = append(out, rollout.Name)
		}
		return out
	}

	forbiddenStore := newStore()
	failingStore := &fakeStore{err: errors.New("database is down")}

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
			name:   "no rollouts exist",
			store:  &fakeStore{},
			method: http.MethodGet,
			url:    projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, decode(t, w))
			},
		},
		{
			name:   "the store fails",
			store:  failingStore,
			method: http.MethodGet,
			url:    projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name:   "lists the Project's rollouts in creation order",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				rollouts := decode(t, w)
				require.Equal(t, []string{"prod.01A.abc", "staging.01A.abc", "prod.01B.abc"}, names(rollouts))
				require.EqualValues(t, 2, rollouts[2].Number)
				require.Equal(
					t,
					[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}, {Name: "us-west"}},
					rollouts[2].Spec.Targets,
				)
				require.Equal(t, kargoapi.PromotionRolloutPhaseRunning, rollouts[2].Status.Phase)
				require.Nil(t, rollouts[2].Status.Targets)
				require.Nil(t, rollouts[2].Status.Summary)
			},
		},
		{
			name:   "narrows the list to a Stage",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc"}, names(decode(t, w)))
			},
		},
		{
			name:  "reports what has become of each Target from the Promotions",
			store: newStore(),
			objects: []client.Object{
				// The newest of two Promotions to us-east counts; us-west has one.
				child("prod.us-east.01A.abc", "prod.01B.abc", "us-east", kargoapi.PromotionPhaseFailed),
				child("prod.us-east.01C.abc", "prod.01B.abc", "us-east", kargoapi.PromotionPhaseSucceeded),
				child("prod.us-west.01A.abc", "prod.01B.abc", "us-west", kargoapi.PromotionPhaseRunning),
				// Another rollout's Promotion, and one to no Target, are ignored.
				child("prod.us-east.00A.abc", "prod.01A.abc", "us-east", kargoapi.PromotionPhaseSucceeded),
				child("prod.00B.abc", "prod.01B.abc", "", kargoapi.PromotionPhaseSucceeded),
			},
			method: http.MethodGet,
			url:    projectURL("?stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				rollouts := decode(t, w)
				require.Len(t, rollouts, 2)
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "us-east", Promotion: "prod.us-east.00A.abc", Phase: kargoapi.PromotionPhaseSucceeded},
				}, rollouts[0].Status.Targets)
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "us-east", Promotion: "prod.us-east.01C.abc", Phase: kargoapi.PromotionPhaseSucceeded},
					{Name: "us-west", Promotion: "prod.us-west.01A.abc", Phase: kargoapi.PromotionPhaseRunning},
				}, rollouts[1].Status.Targets)
				require.Equal(t, &kargoapi.PromotionRolloutSummary{Succeeded: 1, Running: 1}, rollouts[1].Status.Summary)
			},
		},
	})
}
