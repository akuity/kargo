package promotionrollouts

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGet(t *testing.T) {
	t.Parallel()

	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhaseRunning, "us-east")
		return store
	}
	forbiddenStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodGet,
			url:    projectURL("/prod.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       projectURL("/prod.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "not found",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("/prod.01Z.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "the store fails",
			store:  &fakeStore{err: errors.New("database is down")},
			method: http.MethodGet,
			url:    projectURL("/prod.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "found, with what has become of its Targets",
			store: newStore().withPromotions(
				child("prod.us-east.01A.abc", "prod.01A.abc", "us-east", kargoapi.PromotionPhaseRunning),
				child("prod.us-east.00A.abc", "prod.00A.abc", "us-east", kargoapi.PromotionPhaseSucceeded),
			),
			method: http.MethodGet,
			url:    projectURL("/prod.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				rollout := &kargoapi.PromotionRollout{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), rollout))
				require.Equal(t, "prod.01A.abc", rollout.Name)
				require.Equal(t, testProject, rollout.Namespace)
				require.NotEmpty(t, rollout.UID)
				require.NotEmpty(t, rollout.ResourceVersion)
				require.False(t, rollout.CreationTimestamp.IsZero())
				require.EqualValues(t, 1, rollout.Number)
				require.Equal(t, "prod", rollout.Spec.Stage)
				require.Equal(t, []kargoapi.PromotionRolloutTarget{{Name: "us-east"}}, rollout.Spec.Targets)
				require.Equal(t, kargoapi.PromotionRolloutPhaseRunning, rollout.Status.Phase)
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "us-east", Promotion: "prod.us-east.01A.abc", Phase: kargoapi.PromotionPhaseRunning},
				}, rollout.Status.Targets)
				require.Equal(t, &kargoapi.PromotionRolloutSummary{Running: 1}, rollout.Status.Summary)
				require.Len(t, rollout.Status.Conditions, 1)
				require.Equal(t, kargoapi.ConditionTypeReady, rollout.Status.Conditions[0].Type)
			},
		},
		{
			// What has become of the Targets is read from the mirrored
			// Promotions, which the caller must be allowed to list.
			name:      "a caller who may not list Promotions is refused",
			store:     newStore(),
			authorize: forbidPromotions,
			method:    http.MethodGet,
			url:       projectURL("/prod.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
	})
}
