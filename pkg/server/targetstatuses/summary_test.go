package targetstatuses

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/database/targetstatusstore"
)

func TestSummarize(t *testing.T) {
	t.Parallel()

	// counts returns a tally with every State present, as the store's are.
	counts := func(behind int64, states map[targetstatusstore.State]int64) targetstatusstore.Counts {
		tally := targetstatusstore.Counts{States: map[targetstatusstore.State]int64{}, Behind: behind}
		for _, state := range targetstatusstore.States() {
			tally.States[state] = states[state]
		}
		return tally
	}
	stages := []client.Object{
		onFreight(newStage("prod", everything()), "v2"),
		newStage("canary", labeled("tier", "canary")),
		// A Stage that governs nothing yet, and one that never will.
		newStage("apac", labeled("region", "ap")),
		newStage("classic"),
	}
	// Three Targets, one of which no Stage but prod governs, and what the
	// database is taken to have counted for them.
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add("us-west-2", nil)
		store.add("eu-central-1", map[string]string{"tier": "canary"})
		store.add("us-east-1", map[string]string{"tier": "canary"})
		store.summary = targetstatusstore.Summary{
			Targets: counts(2, map[targetstatusstore.State]int64{
				targetstatusstore.StateFailed:   1,
				targetstatusstore.StateBehind:   1,
				targetstatusstore.StateUpToDate: 1,
			}),
			Stages: map[string]targetstatusstore.Counts{
				"prod-uid": counts(1, map[targetstatusstore.State]int64{
					targetstatusstore.StateBehind:   1,
					targetstatusstore.StateUpToDate: 2,
				}),
				"canary-uid": counts(1, map[targetstatusstore.State]int64{
					targetstatusstore.StateFailed:   1,
					targetstatusstore.StateUpToDate: 1,
				}),
			},
		}
		return store
	}
	failingStore := newStore()
	failingStore.err = errors.New("something went wrong")
	scopedStore, narrowedStore := newStore(), newStore()

	runTestCases(t, []testCase{
		{
			name:  "a Project with nothing in it counts nothing, with every State present",
			store: &fakeStore{summary: targetstatusstore.Summary{Targets: counts(0, nil)}},
			url:   projectURL("/target-statuses/summary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
				require.JSONEq(
					t,
					`{
						"targets": 0,
						"states": {
							"Promoting": 0, "Pending": 0, "Failed": 0, "NotAttempted": 0,
							"NeverPromoted": 0, "Behind": 0, "UpToDate": 0
						},
						"behind": 0,
						"stages": []
					}`,
					w.Body.String(),
				)
			},
		},
		{
			name:    "the Project's Targets and each Stage's are counted by State",
			store:   scopedStore,
			objects: stages,
			url:     projectURL("/target-statuses/summary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				summary := decode[Summary](t, w)
				require.Equal(t, int64(3), summary.Targets)
				require.Equal(t, scopedStore.summary.Targets.States, summary.States)
				require.Equal(t, int64(2), summary.Behind)
				// Every target-aware Stage, by name, including the one that
				// governs nothing, which counts nothing.
				require.Equal(
					t,
					[]StageSummary{
						{Stage: "apac", States: counts(0, nil).States},
						{
							Stage:  "canary",
							States: scopedStore.summary.Stages["canary-uid"].States,
							Behind: 1,
						},
						{
							Stage:  "prod",
							States: scopedStore.summary.Stages["prod-uid"].States,
							Behind: 1,
						},
					},
					summary.Stages,
				)
				// The store counted every governed pair.
				require.Len(t, scopedStore.scopes, 1)
				require.Len(t, scopedStore.scopes[0].Pairs, 5)
				require.Equal(
					t,
					map[string][]string{"prod-uid": {"v2"}},
					scopedStore.scopes[0].StageFreight,
				)
			},
		},
		{
			name:    "a Stage narrows the count to the Targets it governs",
			store:   narrowedStore,
			objects: stages,
			url:     projectURL("/target-statuses/summary?stage=canary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				summary := decode[Summary](t, w)
				// The Project still has three Targets.
				require.Equal(t, int64(3), summary.Targets)
				require.Len(t, summary.Stages, 1)
				require.Equal(t, "canary", summary.Stages[0].Stage)
				require.Len(t, narrowedStore.scopes[0].Pairs, 2)
			},
		},
		{
			name:    "a Stage that does not exist is not found",
			store:   newStore(),
			objects: stages,
			url:     projectURL("/target-statuses/summary?stage=no-such-stage"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:    "the store failing fails the request",
			store:   failingStore,
			objects: stages,
			url:     projectURL("/target-statuses/summary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	})
}
