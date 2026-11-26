package promotionrollouts

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/user"
)

func TestAbort(t *testing.T) {
	t.Parallel()

	// The Stage's first rollout is Running and its second is queued behind
	// it; the other Stage's has finished.
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhaseRunning, "us-east")
		store.add(testProject, "prod.01B.abc", "prod", kargoapi.PromotionRolloutPhasePending, "us-east")
		store.add(testProject, "staging.01A.abc", "staging", kargoapi.PromotionRolloutPhaseSucceeded, "us-east")
		return store
	}
	get := func(t *testing.T, store *fakeStore, name string) promotionrolloutstore.Rollout {
		t.Helper()
		rollout, ok := store.project(testProject)[name]
		require.True(t, ok)
		return rollout
	}
	alice := user.KubernetesUser{UserInfo: authnv1.UserInfo{Username: "alice"}}

	forbiddenStore := newStore()
	var authorizations []can.Access
	authorizedStore := newStore()
	runningStore := newStore()
	pendingStore := newStore()
	// Bob asked first.
	againStore := newStore()
	_, err := againStore.Abort(t.Context(), testProject, "prod", 1, "bob")
	require.NoError(t, err)

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodPost,
			url:    stageURL("prod") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:   "an invalid number is refused",
			store:  newStore(),
			method: http.MethodPost,
			url:    stageURL("prod") + "/zero/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodPost,
			url:       stageURL("prod") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
				rollout := get(t, forbiddenStore, "prod.01A.abc")
				require.Nil(t, rollout.AbortRequestedAt)
			},
		},
		{
			name:      "the promote verb on the Stage is what is checked",
			store:     authorizedStore,
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodPost,
			url:       stageURL("prod") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []can.Access{can.Promote().Stage(testProject, "prod")}, authorizations)
			},
		},
		{
			name:   "not found",
			store:  newStore(),
			method: http.MethodPost,
			url:    stageURL("prod") + "/9/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "a finished rollout cannot be aborted",
			store:  newStore(),
			method: http.MethodPost,
			url:    stageURL("staging") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, w.Code)
			},
		},
		{
			name:   "the store fails",
			store:  &fakeStore{err: errors.New("database is down")},
			method: http.MethodPost,
			url:    stageURL("prod") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name:     "a Running rollout records who asked",
			store:    runningStore,
			identity: alice,
			method:   http.MethodPost,
			url:      stageURL("prod") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				rollout := get(t, runningStore, "prod.01A.abc")
				require.NotNil(t, rollout.AbortRequestedAt)
				require.Equal(t, alice.Actor(), rollout.AbortRequestedBy)
				require.Equal(t, string(kargoapi.PromotionRolloutPhaseRunning), rollout.StatusPhase)
				// The one queued behind it is untouched.
				require.Nil(t, get(t, runningStore, "prod.01B.abc").AbortRequestedAt)
			},
		},
		{
			name:     "a Pending rollout ends at once",
			store:    pendingStore,
			identity: alice,
			method:   http.MethodPost,
			url:      stageURL("prod") + "/2/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				rollout := get(t, pendingStore, "prod.01B.abc")
				require.Equal(t, string(kargoapi.PromotionRolloutPhaseAborted), rollout.StatusPhase)
				require.Equal(t, alice.Actor(), rollout.AbortRequestedBy)
				require.Nil(t, get(t, pendingStore, "prod.01A.abc").AbortRequestedAt)
			},
		},
		{
			name:     "asking again keeps the first request",
			store:    againStore,
			identity: alice,
			method:   http.MethodPost,
			url:      stageURL("prod") + "/1/abort",
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, "bob", get(t, againStore, "prod.01A.abc").AbortRequestedBy)
			},
		},
	})
}
