package promotionrollouts

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func TestGetFreightCollection(t *testing.T) {
	t.Parallel()

	images := kargoapi.FreightReference{
		Name:   "images-1",
		Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "images"},
		Images: []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1"}},
	}
	config := kargoapi.FreightReference{
		Name:    "config-1",
		Origin:  kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "config"},
		Commits: []kargoapi.GitCommit{{RepoURL: "https://example.com/config.git", ID: "abc"}},
	}
	recorded := &kargoapi.FreightCollection{}
	recorded.UpdateOrPush(images, config)

	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhaseRunning, "us-east")
		store.add(testProject, "prod.01B.abc", "prod", kargoapi.PromotionRolloutPhasePending, "us-east")
		store.add("other-project", "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhaseRunning, "us-east")
		return store.withCollection(testProject, "prod.01A.abc", recorded)
	}
	url := func(id string) string {
		return projectURL("/" + id + "/freight-collection")
	}
	forbiddenStore := newStore()

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodGet,
			url:    url(rolloutID(testProject, "prod.01A.abc").String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       url(rolloutID(testProject, "prod.01A.abc").String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			// Only the rollout is read, so a caller who may get rollouts but
			// not list Promotions is served.
			name:      "getting rollouts is all it takes",
			store:     newStore(),
			authorize: forbidPromotions,
			method:    http.MethodGet,
			url:       url(rolloutID(testProject, "prod.01A.abc").String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name:   "an invalid id is refused",
			store:  newStore(),
			method: http.MethodGet,
			url:    url("1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:   "an id that is no rollout's is not found",
			store:  newStore(),
			method: http.MethodGet,
			url:    url(uuid.New().String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "another Project's rollout is not found",
			store:  newStore(),
			method: http.MethodGet,
			url:    url(rolloutID("other-project", "prod.01A.abc").String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:   "the store fails",
			store:  &fakeStore{err: errors.New("database is down")},
			method: http.MethodGet,
			url:    url(uuid.New().String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name:   "a rollout that recorded none has an empty collection",
			store:  newStore(),
			method: http.MethodGet,
			url:    url(rolloutID(testProject, "prod.01B.abc").String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				got := &kargoapi.FreightCollection{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), got))
				require.Equal(t, &kargoapi.FreightCollection{}, got)
			},
		},
		{
			name:   "the recorded collection, contents included",
			store:  newStore(),
			method: http.MethodGet,
			url:    url(rolloutID(testProject, "prod.01A.abc").String()),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				got := &kargoapi.FreightCollection{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), got))
				require.Equal(t, recorded, got)
			},
		},
	})
}

// The route asks for get on the Project's rollouts, and nothing else.
func TestGetFreightCollectionAccess(t *testing.T) {
	t.Parallel()
	store := newStoreWithRollout()
	var asked []can.Access
	runTestCases(t, []testCase{{
		name:  "asks for get on rollouts only",
		store: store,
		authorize: func(_ context.Context, access can.Access) error {
			asked = append(asked, access)
			return nil
		},
		method: http.MethodGet,
		url:    projectURL("/" + rolloutID(testProject, "prod.01A.abc").String() + "/freight-collection"),
		assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, []can.Access{can.Get().PromotionRollout(testProject, "")}, asked)
		},
	}})
}

func newStoreWithRollout() *fakeStore {
	store := &fakeStore{}
	store.add(testProject, "prod.01A.abc", "prod", kargoapi.PromotionRolloutPhaseRunning, "us-east")
	return store
}
