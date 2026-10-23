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

	decodeList := func(t *testing.T, w *httptest.ResponseRecorder) *kargoapi.PromotionRolloutList {
		require.Equal(t, http.StatusOK, w.Code)
		list := &kargoapi.PromotionRolloutList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
		require.NotNil(t, list.Items)
		return list
	}
	decode := func(t *testing.T, w *httptest.ResponseRecorder) []kargoapi.PromotionRollout {
		return decodeList(t, w).Items
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
			name:   "lists the Project's rollouts newest first",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decodeList(t, w)
				require.Equal(t, []string{"prod.01B.abc", "staging.01A.abc", "prod.01A.abc"}, names(list.Items))
				require.EqualValues(t, 2, list.Items[0].Number)
				require.Equal(
					t,
					[]kargoapi.PromotionRolloutTarget{{Name: "us-east"}, {Name: "us-west"}},
					list.Items[0].Spec.Targets,
				)
				require.Equal(t, kargoapi.PromotionRolloutPhaseRunning, list.Items[0].Status.Phase)
				require.Nil(t, list.Items[0].Status.Targets)
				require.Nil(t, list.Items[0].Status.Summary)
				// A short page is the last.
				require.Empty(t, list.Continue)
			},
		},
		{
			name:   "narrows the list to a Stage, newest first by number",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"prod.01B.abc", "prod.01A.abc"}, names(decode(t, w)))
			},
		},
		{
			name:   "narrows the list to a Stage's rollouts of one piece of Freight",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=prod&freight=fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"prod.01B.abc", "prod.01A.abc"}, names(decode(t, w)))
			},
		},
		{
			name:   "a Freight the Stage never promoted has no rollouts",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=prod&freight=other"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, decode(t, w))
			},
		},
		{
			name:   "freight requires a Stage",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?freight=fake-freight"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "a full page says where the next one starts",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?limit=2"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decodeList(t, w)
				require.Equal(t, []string{"prod.01B.abc", "staging.01A.abc"}, names(list.Items))
				require.Equal(t, "staging.01A.abc", list.Continue)
			},
		},
		{
			name:   "continues from where the previous page ended",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?limit=2&continue=staging.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decodeList(t, w)
				require.Equal(t, []string{"prod.01A.abc"}, names(list.Items))
				require.Empty(t, list.Continue)
			},
		},
		{
			name:   "an unknown continue token is refused",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?continue=nope"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "an invalid limit is refused",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?limit=0"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "lists a Stage's rollouts after a number, oldest first",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=prod&after=0"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc"}, names(decode(t, w)))
			},
		},
		{
			name:   "after excludes the number given",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?stage=prod&after=1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"prod.01B.abc"}, names(decode(t, w)))
			},
		},
		{
			name:   "after requires a Stage",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?after=0"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			// prod is promoting through prod.01B and last finished prod.01A;
			// staging is promoting through staging.01A and has finished none.
			name:   "lists each Stage's latest rollouts",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?latest=true"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decodeList(t, w)
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc", "staging.01A.abc"}, names(list.Items))
				require.Empty(t, list.Continue)
			},
		},
		{
			name:   "lists the named Stage's latest rollouts",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?latest=true&stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc"}, names(decode(t, w)))
			},
		},
		{
			name:   "the latest rollouts are not paged",
			store:  newStore(),
			method: http.MethodGet,
			url:    projectURL("?latest=true&continue=prod.01A.abc"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "reports what has become of each Target from the Promotions",
			store: newStore().withPromotions(
				// The newest of two Promotions to us-east counts; us-west has one.
				child("prod.us-east.01A.abc", "prod.01B.abc", "us-east", kargoapi.PromotionPhaseFailed),
				child("prod.us-east.01C.abc", "prod.01B.abc", "us-east", kargoapi.PromotionPhaseSucceeded),
				child("prod.us-west.01A.abc", "prod.01B.abc", "us-west", kargoapi.PromotionPhaseRunning),
				// Another rollout's Promotion, and one to no Target, are ignored.
				child("prod.us-east.00A.abc", "prod.01A.abc", "us-east", kargoapi.PromotionPhaseSucceeded),
				child("prod.00B.abc", "prod.01B.abc", "", kargoapi.PromotionPhaseSucceeded),
			),
			method: http.MethodGet,
			url:    projectURL("?stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				rollouts := decode(t, w)
				require.Len(t, rollouts, 2)
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "us-east", Promotion: "prod.us-east.01C.abc", Phase: kargoapi.PromotionPhaseSucceeded},
					{Name: "us-west", Promotion: "prod.us-west.01A.abc", Phase: kargoapi.PromotionPhaseRunning},
				}, rollouts[0].Status.Targets)
				require.Equal(t, &kargoapi.PromotionRolloutSummary{Succeeded: 1, Running: 1}, rollouts[0].Status.Summary)
				require.Equal(t, []kargoapi.PromotionRolloutTargetStatus{
					{Name: "us-east", Promotion: "prod.us-east.00A.abc", Phase: kargoapi.PromotionPhaseSucceeded},
				}, rollouts[1].Status.Targets)
			},
		},
		{
			// What has become of the Targets is read from the mirrored
			// Promotions, which the caller must be allowed to list.
			name:      "a caller who may not list Promotions is refused",
			store:     newStore(),
			authorize: forbidPromotions,
			method:    http.MethodGet,
			url:       projectURL("?stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
			},
		},
		{
			// With no rollouts there are no Promotions to read.
			name:      "an empty list does not ask about Promotions",
			store:     &fakeStore{},
			authorize: forbidPromotions,
			method:    http.MethodGet,
			url:       projectURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, decode(t, w))
			},
		},
	})
}
