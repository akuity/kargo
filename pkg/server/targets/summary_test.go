package targets

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

func summariesURL(query string) string {
	return "/v1beta1/projects/" + testProject + "/target-summaries" + query
}

func TestListSummaries(t *testing.T) {
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
	region := func(value string) metav1.LabelSelector {
		return metav1.LabelSelector{MatchLabels: map[string]string{"region": value}}
	}
	stages := []client.Object{
		newStage("fleet", region("us")),
		newStage("europe", region("eu")),
		newStage("classic"),
		newStage("broken", metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "region", Operator: "Bogus"}},
		}),
	}
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add(testProject, "us-west-2", map[string]string{"region": "us"})
		store.add(testProject, "eu-central-1", map[string]string{"region": "eu", "tier": "canary"})
		store.add(testProject, "us-east-1", map[string]string{"region": "us", "tier": "canary"})
		store.add(testProject, "ap-south-1", map[string]string{"region": "ap"})
		store.addFreight("us-east-1", "fleet", "f1")
		// A Stage that no longer governs a Target says nothing about it.
		store.addFreight("us-east-1", "europe", "f9")
		store.addPromotion("us-east-1", "fleet", "p1", "Succeeded", 1)
		store.addPromotion("us-east-1", "fleet", "p2", "Running", 2)
		store.addPromotion("us-east-1", "europe", "p0", "Succeeded", 3)
		store.addPromotion("eu-central-1", "europe", "p3", "Failed", 4)
		return store
	}
	list := func(t *testing.T, w *httptest.ResponseRecorder) []TargetSummary {
		require.Equal(t, http.StatusOK, w.Code)
		summaries := &TargetSummaryList{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), summaries))
		require.NotNil(t, summaries.Items)
		return summaries.Items
	}
	names := func(summaries []TargetSummary) []string {
		out := make([]string, 0, len(summaries))
		for _, summary := range summaries {
			out = append(out, summary.Name)
		}
		return out
	}

	var authorizations []can.Access
	forbiddenStore := newStore()
	emptyStore := &fakeStore{}

	runTestCases(t, []testCase{
		{
			name:   "database is not configured",
			method: http.MethodGet,
			url:    summariesURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotImplemented, w.Code)
			},
		},
		{
			name:      "not authorized",
			store:     forbiddenStore,
			authorize: forbidEverything,
			method:    http.MethodGet,
			url:       summariesURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Zero(t, forbiddenStore.calls)
			},
		},
		{
			name:   "no Targets reads no Freight",
			store:  emptyStore,
			method: http.MethodGet,
			url:    summariesURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, list(t, w))
				require.Zero(t, emptyStore.freightReads)
			},
		},
		{
			name:      "summarizes each Target under the Stages that govern it",
			store:     newStore(),
			objects:   stages,
			authorize: recordAuthorizations(&authorizations),
			method:    http.MethodGet,
			url:       summariesURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []can.Access{can.List().Target(testProject, "")}, authorizations)
				summaries := list(t, w)
				require.Equal(
					t,
					[]string{"ap-south-1", "eu-central-1", "us-east-1", "us-west-2"},
					names(summaries),
				)
				// Governed by nothing.
				require.Equal(t, map[string]string{"region": "ap"}, summaries[0].Labels)
				require.Nil(t, summaries[0].Stages)
				// Promoted to, never successfully.
				europe := summaries[1].Stages["europe"]
				require.Len(t, summaries[1].Stages, 1)
				require.Empty(t, europe.CollectionID)
				require.Empty(t, europe.Freight)
				require.Equal(t, "p3", europe.LastPromotion.Name)
				require.Equal(t, "Failed", europe.LastPromotion.Phase)
				require.Equal(t, "europe", europe.LastPromotion.Stage)
				require.Equal(t, "f-p3", europe.LastPromotion.Freight)
				// Running on the Freight of an earlier success.
				require.Len(t, summaries[2].Stages, 1)
				fleet := summaries[2].Stages["fleet"]
				require.Equal(t, collectionID("f1"), fleet.CollectionID)
				require.Equal(t, []string{"f1"}, fleet.Freight)
				require.Equal(t, "p2", fleet.LastPromotion.Name)
				require.Equal(t, "Running", fleet.LastPromotion.Phase)
				require.Nil(t, fleet.LastPromotion.FinishedAt)
				// Governed, never promoted to.
				require.Equal(t, map[string]TargetStageSummary{"fleet": {}}, summaries[3].Stages)
			},
		},
		{
			name:    "a Stage filter keeps the Targets it governs",
			store:   newStore(),
			objects: stages,
			method:  http.MethodGet,
			url:     summariesURL("?stage=fleet"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				summaries := list(t, w)
				require.Equal(t, []string{"us-east-1", "us-west-2"}, names(summaries))
				require.Len(t, summaries[0].Stages, 1)
			},
		},
		{
			name:    "a classic Stage governs no Targets",
			store:   newStore(),
			objects: stages,
			method:  http.MethodGet,
			url:     summariesURL("?stage=classic"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Empty(t, list(t, w))
			},
		},
		{
			name:    "a Stage filter names a Stage that does not exist",
			store:   newStore(),
			objects: stages,
			method:  http.MethodGet,
			url:     summariesURL("?stage=missing"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, w.Code)
			},
		},
		{
			name:    "a Stage filter names a Stage whose selectors do not parse",
			store:   newStore(),
			objects: stages,
			method:  http.MethodGet,
			url:     summariesURL("?stage=broken"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name:    "filters by label selector",
			store:   newStore(),
			objects: stages,
			method:  http.MethodGet,
			url:     summariesURL("?labelSelector=tier%3Dcanary"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, []string{"eu-central-1", "us-east-1"}, names(list(t, w)))
			},
		},
		{
			name:   "rejects a malformed label selector",
			store:  newStore(),
			method: http.MethodGet,
			url:    summariesURL("?labelSelector=region%3D%3D%3D"),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "a failure to read current Freight fails the list",
			store: func() *fakeStore {
				store := newStore()
				store.freightErr = errors.New("offline")
				return store
			}(),
			objects: stages,
			method:  http.MethodGet,
			url:     summariesURL(""),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
	})
}
