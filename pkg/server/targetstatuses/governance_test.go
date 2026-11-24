package targetstatuses

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func TestGovern(t *testing.T) {
	t.Parallel()

	// A selector no Kubernetes label selector may be.
	unparseable := metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{
		Key:      "region",
		Operator: "Sometimes",
	}}}
	otherProject := newStage("elsewhere", everything())
	otherProject.Namespace = "other-project"

	// Three Targets, added out of name order.
	newStore := func() *fakeStore {
		store := &fakeStore{}
		store.add("us-west-2", map[string]string{"region": "us"})
		store.add("eu-central-1", map[string]string{"region": "eu", "tier": "canary"})
		store.add("us-east-1", map[string]string{"region": "us", "tier": "canary"})
		return store
	}
	// pairs lists a fleet's pairs as "<target>/<stage name>".
	pairs := func(t *testing.T, f *fleet) []string {
		out := make([]string, len(f.pairs))
		for i, pair := range f.pairs {
			require.Equal(t, pair.TargetName, f.targets[pair.TargetID].Name)
			out[i] = pair.TargetName + "/" + f.stageName(pair.StageID)
		}
		return out
	}

	testCases := []struct {
		name    string
		store   *fakeStore
		objects []client.Object
		stage   string
		assert  func(*testing.T, *fleet, error)
	}{
		{
			name:  "a Project with no Stages governs nothing",
			store: newStore(),
			assert: func(t *testing.T, f *fleet, err error) {
				require.NoError(t, err)
				require.Empty(t, f.stages)
				require.Empty(t, f.pairs)
				// Its Targets are still its Targets.
				require.Len(t, f.targets, 3)
			},
		},
		{
			name:  "each Target is paired with every Stage whose selectors it matches",
			store: newStore(),
			objects: []client.Object{
				newStage("prod", everything()),
				// Either selector is enough, and matching both pairs once.
				newStage("canary", labeled("tier", "canary"), labeled("region", "eu")),
				newStage("apac", labeled("region", "ap")),
				// A classic Stage, and another Project's, govern none of these.
				newStage("classic"),
				otherProject,
			},
			assert: func(t *testing.T, f *fleet, err error) {
				require.NoError(t, err)
				// By Target name and then Stage name.
				require.Equal(
					t,
					[]string{
						"eu-central-1/canary",
						"eu-central-1/prod",
						"us-east-1/canary",
						"us-east-1/prod",
						"us-west-2/prod",
					},
					pairs(t, f),
				)
				// A Stage that governs nothing is still one of the Stages.
				names := make([]string, len(f.stages))
				for i, stage := range f.stages {
					names[i] = stage.name
					require.Equal(t, stage.name+"-uid", stage.id)
				}
				require.Equal(t, []string{"apac", "canary", "prod"}, names)
				require.Empty(t, f.stageName("no-such-uid"))
			},
		},
		{
			name:  "a Stage's current Freight goes with it",
			store: newStore(),
			objects: []client.Object{
				onFreight(newStage("prod", everything()), "v2", "c1"),
				newStage("staging", everything()),
			},
			assert: func(t *testing.T, f *fleet, err error) {
				require.NoError(t, err)
				scope := f.scope(f.pairs[:1])
				require.Equal(t, f.pairs[:1], scope.Pairs)
				// A Stage that is on nothing yet is left out.
				require.Len(t, scope.StageFreight, 1)
				require.ElementsMatch(t, []string{"v2", "c1"}, scope.StageFreight["prod-uid"])
			},
		},
		{
			name:  "a Stage whose selectors do not parse governs nothing and hides nothing",
			store: newStore(),
			objects: []client.Object{
				newStage("broken", unparseable),
				newStage("prod", everything()),
			},
			assert: func(t *testing.T, f *fleet, err error) {
				require.NoError(t, err)
				require.Len(t, f.stages, 1)
				require.Equal(
					t,
					[]string{"eu-central-1/prod", "us-east-1/prod", "us-west-2/prod"},
					pairs(t, f),
				)
			},
		},
		{
			name:  "a named Stage narrows governance to itself",
			store: newStore(),
			objects: []client.Object{
				newStage("prod", everything()),
				newStage("canary", labeled("tier", "canary")),
			},
			stage: "canary",
			assert: func(t *testing.T, f *fleet, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"eu-central-1/canary", "us-east-1/canary"}, pairs(t, f))
				require.Len(t, f.stages, 1)
				require.Len(t, f.targets, 3)
			},
		},
		{
			name:    "a named classic Stage governs nothing",
			store:   newStore(),
			objects: []client.Object{newStage("classic")},
			stage:   "classic",
			assert: func(t *testing.T, f *fleet, err error) {
				require.NoError(t, err)
				require.Empty(t, f.stages)
				require.Empty(t, f.pairs)
			},
		},
		{
			name:  "a named Stage that does not exist is not found",
			store: newStore(),
			stage: "no-such-stage",
			assert: func(t *testing.T, _ *fleet, err error) {
				require.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name:    "a named Stage whose selectors do not parse is a bad request",
			store:   newStore(),
			objects: []client.Object{newStage("broken", unparseable)},
			stage:   "broken",
			assert: func(t *testing.T, _ *fleet, err error) {
				var httpErr *libhttp.HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, http.StatusBadRequest, httpErr.Code())
			},
		},
		{
			name:    "the store failing fails governance",
			store:   &fakeStore{err: errors.New("something went wrong")},
			objects: []client.Object{newStage("prod", everything())},
			assert: func(t *testing.T, _ *fleet, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			h := New(testCase.store, nil, newKube(t, testCase.objects...))
			f, err := h.govern(context.Background(), testProject, testCase.stage)
			testCase.assert(t, f, err)
		})
	}
}

func TestCurrentFreight(t *testing.T) {
	t.Parallel()
	require.Empty(t, currentFreight(&kargoapi.Stage{}))
	stage := onFreight(newStage("prod", everything()), "v2", "c1")
	// Only the newest collection is current.
	older := &kargoapi.FreightCollection{}
	older.UpdateOrPush(kargoapi.FreightReference{
		Name:   "v1",
		Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "old"},
	})
	stage.Status.FreightHistory = append(stage.Status.FreightHistory, older)
	require.ElementsMatch(t, []string{"v2", "c1"}, currentFreight(stage))
}

func TestEntryCursorSearch(t *testing.T) {
	t.Parallel()
	f := &fleet{
		stages: []governingStage{{id: "canary-uid", name: "canary"}, {id: "prod-uid", name: "prod"}},
		pairs: []targetstatusstore.Pair{
			{TargetName: "a", StageID: "canary-uid"},
			{TargetName: "a", StageID: "prod-uid"},
			{TargetName: "c", StageID: "prod-uid"},
		},
	}
	testCases := []struct {
		name     string
		cursor   entryCursor
		expected int
	}{
		{name: "before everything", cursor: entryCursor{Target: "A", Stage: "prod"}, expected: 0},
		{name: "at the first pair", cursor: entryCursor{Target: "a", Stage: "canary"}, expected: 1},
		{name: "at the last pair of a Target", cursor: entryCursor{Target: "a", Stage: "prod"}, expected: 2},
		{
			name:     "at a Stage that no longer governs the Target",
			cursor:   entryCursor{Target: "a", Stage: "dev"},
			expected: 1,
		},
		{
			name:     "at a Target that is gone",
			cursor:   entryCursor{Target: "b", Stage: "prod"},
			expected: 2,
		},
		{name: "at the last pair", cursor: entryCursor{Target: "c", Stage: "prod"}, expected: 3},
		{name: "after everything", cursor: entryCursor{Target: "z", Stage: "prod"}, expected: 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, testCase.cursor.search(f))
		})
	}
}
