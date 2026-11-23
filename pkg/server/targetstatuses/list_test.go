package targetstatuses

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
)

func TestList(t *testing.T) {
	t.Parallel()

	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := created.Add(time.Minute)

	// Three Targets under prod, two of them under canary as well: five
	// entries, each with a different story.
	// Each use gets Stages of its own: a fake client keeps the objects it is
	// built with, and the subtests below run in parallel.
	newStages := func() []client.Object {
		return []client.Object{
			onFreight(newStage("prod", everything()), "v2"),
			newStage("canary", labeled("tier", "canary")),
			newStage("classic"),
		}
	}
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add("us-west-2", nil)
		store.add("eu-central-1", map[string]string{"tier": "canary"})
		store.add("us-east-1", map[string]string{"tier": "canary", "region": "us"})
		store.set("eu-central-1", "prod-uid", targetstatusstore.Entry{
			State:  targetstatusstore.StateBehind,
			Behind: true,
			Last: &targetstatusstore.Promotion{
				Name:       "prod.abc",
				Freight:    "v1",
				Phase:      "Succeeded",
				Rollout:    3,
				CreatedAt:  created,
				FinishedAt: &finished,
			},
			Freight: []targetstatusstore.Freight{
				{OriginKind: "Warehouse", OriginName: "app", Name: "v1"},
				{OriginKind: "Warehouse", OriginName: "config", Name: "c1"},
			},
		})
		store.set("us-east-1", "canary-uid", targetstatusstore.Entry{
			State:    targetstatusstore.StatePending,
			InFlight: true,
			Rollout:  7,
		})
		store.set("us-east-1", "prod-uid", targetstatusstore.Entry{
			State: targetstatusstore.StateFailed,
			Last: &targetstatusstore.Promotion{
				Name:      "prod.def",
				Freight:   "v2",
				Phase:     "Errored",
				CreatedAt: created,
			},
		})
		return store
	}
	// keys lists a page's entries as "<target>/<stage>".
	keys := func(list EntryList) []string {
		out := make([]string, len(list.Items))
		for i, item := range list.Items {
			out[i] = item.Target + "/" + item.Stage
		}
		return out
	}
	all := []string{
		"eu-central-1/canary",
		"eu-central-1/prod",
		"us-east-1/canary",
		"us-east-1/prod",
		"us-west-2/prod",
	}

	failingStore := newStore()
	failingStore.err = errors.New("something went wrong")
	scopedStore, cappedStore, filteredStore := newStore(), newStore(), newStore()

	runTestCases(t, []testCase{
		{
			name:  "a Project with no Targets has no entries",
			store: &fakeStore{},
			url:   projectURL("/target-statuses"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				// An empty list, not a missing one.
				require.JSONEq(t, `{"items":[]}`, w.Body.String())
			},
		},
		{
			name:    "every Target is listed with every Stage that governs it",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				list := decode[EntryList](t, w)
				require.Equal(t, all, keys(list))
				require.Empty(t, list.Continue)

				// Nothing on record: never promoted to.
				require.Equal(
					t,
					Entry{
						Target:  "eu-central-1",
						Labels:  map[string]string{"tier": "canary"},
						Stage:   "canary",
						State:   targetstatusstore.StateNeverPromoted,
						Freight: []Freight{},
					},
					list.Items[0],
				)
				// Freight per origin, and the last Promotion, which names the
				// entry's Stage.
				require.Equal(
					t,
					Entry{
						Target: "eu-central-1",
						Labels: map[string]string{"tier": "canary"},
						Stage:  "prod",
						State:  targetstatusstore.StateBehind,
						Behind: true,
						Freight: []Freight{
							{
								Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "app"},
								Name:   "v1",
							},
							{
								Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "config"},
								Name:   "c1",
							},
						},
						LastPromotion: &Promotion{
							Name:       "prod.abc",
							Stage:      "prod",
							Freight:    "v1",
							Phase:      kargoapi.PromotionPhaseSucceeded,
							Rollout:    3,
							CreatedAt:  created,
							FinishedAt: &finished,
						},
					},
					list.Items[1],
				)
				// Waiting on a rollout, which the entry names.
				require.Equal(t, targetstatusstore.StatePending, list.Items[2].State)
				require.True(t, list.Items[2].InFlight)
				require.Equal(t, int64(7), list.Items[2].Rollout)
				require.Nil(t, list.Items[2].LastPromotion)
				// A Promotion that has not finished has no finish time.
				require.Equal(t, kargoapi.PromotionPhaseErrored, list.Items[3].LastPromotion.Phase)
				require.Nil(t, list.Items[3].LastPromotion.FinishedAt)
				// A Target without labels has none.
				require.Nil(t, list.Items[4].Labels)
			},
		},
		{
			name:    "absent fields are left out of the response",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses?stage=canary&limit=1"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				list := decode[EntryList](t, w)
				require.JSONEq(
					t,
					`{
						"items": [{
							"target": "eu-central-1",
							"labels": {"tier": "canary"},
							"stage": "canary",
							"state": "NeverPromoted",
							"inFlight": false,
							"behind": false,
							"freight": []
						}],
						"continue": "`+list.Continue+`"
					}`,
					w.Body.String(),
				)
			},
		},
		{
			name:    "the store is asked about the governed pairs and the Stages' Freight",
			store:   scopedStore,
			objects: newStages(),
			url:     projectURL("/target-statuses"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Len(t, scopedStore.scopes, 1)
				scope := scopedStore.scopes[0]
				require.Len(t, scope.Pairs, 5)
				require.Equal(t, "eu-central-1", scope.Pairs[0].TargetName)
				require.Equal(t, "canary-uid", scope.Pairs[0].StageID)
				require.Equal(t, map[string][]string{"prod-uid": {"v2"}}, scope.StageFreight)
				require.Empty(t, scopedStore.states[0])
				require.Equal(t, []int32{defaultPageSize}, scopedStore.limits)
			},
		},
		{
			name:    "a Stage narrows the list to the Targets it governs",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses?stage=canary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(
					t,
					[]string{"eu-central-1/canary", "us-east-1/canary"},
					keys(decode[EntryList](t, w)),
				)
			},
		},
		{
			name:    "a classic Stage has no entries",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses?stage=classic"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, decode[EntryList](t, w).Items)
			},
		},
		{
			name:    "a Stage that does not exist is not found",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses?stage=no-such-stage"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:    "states narrow the list to entries in any of them",
			store:   filteredStore,
			objects: newStages(),
			url:     projectURL("/target-statuses?state=Failed&state=NeverPromoted"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(
					t,
					[]string{"eu-central-1/canary", "us-east-1/prod", "us-west-2/prod"},
					keys(decode[EntryList](t, w)),
				)
				require.Equal(
					t,
					[][]targetstatusstore.State{{
						targetstatusstore.StateFailed,
						targetstatusstore.StateNeverPromoted,
					}},
					filteredStore.states,
				)
			},
		},
		{
			name:    "a state outside the vocabulary is refused before anything is read",
			store:   scopedStore,
			objects: newStages(),
			url:     projectURL("/target-statuses?state=Healthy"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), `invalid state \"Healthy\"`)
				// Only the earlier case's two reads.
				require.Equal(t, 2, scopedStore.calls)
			},
		},
		{
			name:    "a limit that is not a positive number is refused",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses?limit=0"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:    "a limit is capped",
			store:   cappedStore,
			objects: newStages(),
			url:     projectURL("/target-statuses?limit=100000"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []int32{maxPageSize}, cappedStore.limits)
			},
		},
		{
			name:    "a continue token that is not one is refused",
			store:   newStore(),
			objects: newStages(),
			url:     projectURL("/target-statuses?continue=not-a-token"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
				require.Contains(t, w.Body.String(), "invalid continue token")
			},
		},
		{
			name:    "the store failing fails the request",
			store:   failingStore,
			objects: newStages(),
			url:     projectURL("/target-statuses"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	})

	// page reads one page through a router that keeps its store and Stages
	// between requests.
	page := func(t *testing.T, store *fakeStore, query url.Values) EntryList {
		w := httptest.NewRecorder()
		newRouter(store, forbid("nothing"), newKube(t, newStages()...)).ServeHTTP(
			w,
			httptest.NewRequest(http.MethodGet, projectURL("/target-statuses?"+query.Encode()), nil),
		)
		return decode[EntryList](t, w)
	}
	// readAll pages until a page has no continue token.
	readAll := func(t *testing.T, store *fakeStore, query url.Values) ([]string, int) {
		var read []string
		pages := 0
		for {
			list := page(t, store, query)
			pages++
			read = append(read, keys(list)...)
			if list.Continue == "" {
				return read, pages
			}
			require.Len(t, list.Items, 2)
			query.Set(queryContinue, list.Continue)
		}
	}

	t.Run("pages follow one another without gap or overlap", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		read, pages := readAll(t, store, url.Values{queryLimit: {"2"}})
		require.Equal(t, all, read)
		require.Equal(t, 3, pages)
		// Each page asks the store only about the pairs after the last.
		require.Len(t, store.scopes[0].Pairs, 5)
		require.Len(t, store.scopes[1].Pairs, 3)
		require.Len(t, store.scopes[2].Pairs, 1)
		// A page that happens to be full is followed by an empty one.
		read, pages = readAll(t, newStore(), url.Values{queryLimit: {"2"}, queryStage: {"canary"}})
		require.Equal(t, []string{"eu-central-1/canary", "us-east-1/canary"}, read)
		require.Equal(t, 2, pages)
	})

	t.Run("filtered pages follow one another too", func(t *testing.T) {
		t.Parallel()
		read, pages := readAll(t, newStore(), url.Values{
			queryLimit: {"2"},
			queryState: {"NeverPromoted", "Failed"},
		})
		require.Equal(t, []string{"eu-central-1/canary", "us-east-1/prod", "us-west-2/prod"}, read)
		require.Equal(t, 2, pages)
	})

	t.Run("a page continues from where the last ended, whatever has gone since", func(t *testing.T) {
		t.Parallel()
		store := newStore()
		first := page(t, store, url.Values{queryLimit: {"2"}})
		require.Equal(t, all[:2], keys(first))
		// The Target the page ended at is deleted before the next is read.
		store.targets[testProject] = store.targets[testProject][:1]
		store.add("us-east-1", map[string]string{"tier": "canary"})
		next := page(t, store, url.Values{queryLimit: {"2"}, queryContinue: {first.Continue}})
		require.Equal(t, []string{"us-east-1/canary", "us-east-1/prod"}, keys(next))
	})
}

func TestParseStates(t *testing.T) {
	t.Parallel()
	states, err := parseStates(nil)
	require.NoError(t, err)
	require.Empty(t, states)

	var every []string
	for _, state := range targetstatusstore.States() {
		every = append(every, string(state))
	}
	states, err = parseStates(every)
	require.NoError(t, err)
	require.Equal(t, targetstatusstore.States(), states)

	// The vocabulary is exact: nothing is matched loosely.
	for _, raw := range []string{"", "failed", "Failed,Behind", "Succeeded"} {
		_, err = parseStates([]string{"Failed", raw})
		require.Error(t, err, raw)
	}
}
