package targetstatuses

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
)

func TestListPromotions(t *testing.T) {
	t.Parallel()

	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := created.Add(time.Minute)
	// Three Promotions to one Target, newest first as the store returns them.
	newStore := func() *fakeStore {
		return &fakeStore{promotions: map[string][]targetstatusstore.Promotion{
			"us-east-1": {
				{
					ID:        "id-3",
					Name:      "prod.ghi",
					Stage:     "prod",
					Freight:   "v3",
					Phase:     "Running",
					Rollout:   9,
					CreatedAt: created.Add(2 * time.Hour),
				},
				{
					ID:         "id-2",
					Name:       "canary.def",
					Stage:      "canary",
					Freight:    "v2",
					Phase:      "Failed",
					CreatedAt:  created.Add(time.Hour),
					FinishedAt: &finished,
				},
				{
					ID:         "id-1",
					Name:       "prod.abc",
					Stage:      "prod",
					Freight:    "v1",
					Phase:      "Succeeded",
					Rollout:    8,
					CreatedAt:  created,
					FinishedAt: &finished,
				},
			},
		}}
	}
	names := func(list PromotionList) []string {
		out := make([]string, len(list.Items))
		for i, item := range list.Items {
			out[i] = item.Name
		}
		return out
	}
	failingStore := newStore()
	failingStore.err = errors.New("something went wrong")
	optionsStore, cappedStore := newStore(), newStore()

	runTestCases(t, []testCase{
		{
			name:  "a Target with no Promotions has an empty history",
			store: newStore(),
			url:   projectURL("/targets/no-such-target/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.JSONEq(t, `{"items":[]}`, w.Body.String())
			},
		},
		{
			name:  "a Target's Promotions are served newest first",
			store: newStore(),
			url:   projectURL("/targets/us-east-1/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decode[PromotionList](t, w)
				require.Equal(t, []string{"prod.ghi", "canary.def", "prod.abc"}, names(list))
				require.Empty(t, list.Continue)
				require.Equal(
					t,
					Promotion{
						Name:      "prod.ghi",
						Stage:     "prod",
						Freight:   "v3",
						Phase:     kargoapi.PromotionPhaseRunning,
						Rollout:   9,
						CreatedAt: created.Add(2 * time.Hour),
					},
					list.Items[0],
				)
				require.Equal(t, &finished, list.Items[1].FinishedAt)
				require.Zero(t, list.Items[1].Rollout)
			},
		},
		{
			name:  "absent fields are left out of the response",
			store: newStore(),
			url:   projectURL("/targets/us-east-1/promotions?limit=1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decode[PromotionList](t, w)
				require.JSONEq(
					t,
					`{
						"items": [{
							"name": "prod.ghi",
							"stage": "prod",
							"freight": "v3",
							"phase": "Running",
							"rollout": 9,
							"createdAt": "2020-01-02T05:04:05Z"
						}],
						"continue": "`+list.Continue+`"
					}`,
					w.Body.String(),
				)
			},
		},
		{
			name:  "the Stage and the default limit are passed to the store",
			store: optionsStore,
			url:   projectURL("/targets/us-east-1/promotions?stage=prod"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(
					t,
					[]targetstatusstore.PromotionListOptions{{Stage: "prod", Limit: defaultPageSize}},
					optionsStore.promotionOpts,
				)
			},
		},
		{
			name:  "a limit is capped",
			store: cappedStore,
			url:   projectURL("/targets/us-east-1/promotions?limit=100000"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, int32(maxPageSize), cappedStore.promotionOpts[0].Limit)
			},
		},
		{
			name:  "a limit that is not a positive number is refused",
			store: newStore(),
			url:   projectURL("/targets/us-east-1/promotions?limit=none"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:  "a continue token that is not one is refused",
			store: newStore(),
			url:   projectURL("/targets/us-east-1/promotions?continue=not-a-token"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:  "the store failing fails the request",
			store: failingStore,
			url:   projectURL("/targets/us-east-1/promotions"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	})

	t.Run("pages follow one another without gap or overlap", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		router := newRouter(store, forbid("nothing"))
		query := url.Values{queryLimit: {"2"}}
		var read []string
		for pages := 1; ; pages++ {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(
				http.MethodGet,
				projectURL("/targets/us-east-1/promotions?"+query.Encode()),
				nil,
			))
			list := decode[PromotionList](t, w)
			read = append(read, names(list)...)
			if list.Continue == "" {
				require.Equal(t, 2, pages)
				break
			}
			query.Set(queryContinue, list.Continue)
		}
		require.Equal(t, []string{"prod.ghi", "canary.def", "prod.abc"}, read)
		// The second page was read from before the first one's last Promotion.
		require.Nil(t, store.promotionOpts[0].Before)
		require.Equal(
			t,
			&targetstatusstore.PromotionCursor{CreatedAt: created.Add(time.Hour), ID: "id-2"},
			store.promotionOpts[1].Before,
		)
	})
}

func TestPresentPromotion(t *testing.T) {
	t.Parallel()
	// The driver hands times back in the server's zone; the API serves UTC.
	zone := time.FixedZone("elsewhere", -5*60*60)
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, zone)
	finished := created.Add(time.Minute)
	promotion := presentPromotion(targetstatusstore.Promotion{
		ID:         "id-1",
		Name:       "prod.abc",
		Stage:      "prod",
		Freight:    "v1",
		Phase:      "Succeeded",
		Rollout:    3,
		CreatedAt:  created,
		FinishedAt: &finished,
	})
	require.Equal(
		t,
		Promotion{
			Name:       "prod.abc",
			Stage:      "prod",
			Freight:    "v1",
			Phase:      kargoapi.PromotionPhaseSucceeded,
			Rollout:    3,
			CreatedAt:  time.Date(2020, 1, 2, 8, 4, 5, 0, time.UTC),
			FinishedAt: ptr(time.Date(2020, 1, 2, 8, 5, 5, 0, time.UTC)),
		},
		promotion,
	)
	// The caller's time is left as it was.
	require.Equal(t, zone, finished.Location())

	require.Nil(t, presentPromotion(targetstatusstore.Promotion{CreatedAt: created}).FinishedAt)
}

func ptr[T any](value T) *T { return &value }
