package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestTargetSelectorsForStage(t *testing.T) {
	testCases := []struct {
		name   string
		stage  *kargoapi.Stage
		assert func(*testing.T, []labels.Selector, error)
	}{
		{
			name:  "classic Stage governs no Targets",
			stage: &kargoapi.Stage{},
			assert: func(t *testing.T, selectors []labels.Selector, err error) {
				require.NoError(t, err)
				require.Nil(t, selectors)
			},
		},
		{
			name: "target-aware Stage with an empty selector list",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{}},
				},
			},
			assert: func(t *testing.T, selectors []labels.Selector, err error) {
				require.NoError(t, err)
				require.NotNil(t, selectors)
				require.Empty(t, selectors)
			},
		},
		{
			name: "malformed selector",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{{
						MatchExpressions: []metav1.LabelSelectorRequirement{{
							Key:      "region",
							Operator: "Bogus",
						}},
					}}},
				},
			},
			assert: func(t *testing.T, _ []labels.Selector, err error) {
				require.ErrorContains(t, err, "error parsing target selector 0")
			},
		},
		{
			name: "parses every selector",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{
						{MatchLabels: map[string]string{"region": "us"}},
						{},
					}},
				},
			},
			assert: func(t *testing.T, selectors []labels.Selector, err error) {
				require.NoError(t, err)
				require.Len(t, selectors, 2)
				require.True(t, selectors[0].Matches(labels.Set{"region": "us"}))
				require.False(t, selectors[0].Matches(labels.Set{"region": "eu"}))
				// The empty selector matches everything.
				require.True(t, selectors[1].Matches(labels.Set{}))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			selectors, err := TargetSelectorsForStage(testCase.stage)
			testCase.assert(t, selectors, err)
		})
	}
}

func TestAnySelectorMatches(t *testing.T) {
	us := labels.SelectorFromSet(labels.Set{"region": "us"})
	eu := labels.SelectorFromSet(labels.Set{"region": "eu"})

	testCases := []struct {
		name      string
		selectors []labels.Selector
		labels    map[string]string
		expected  bool
	}{
		{
			name:     "no selectors match nothing",
			labels:   map[string]string{"region": "us"},
			expected: false,
		},
		{
			name:      "first selector matches",
			selectors: []labels.Selector{us, eu},
			labels:    map[string]string{"region": "us"},
			expected:  true,
		},
		{
			name:      "later selector matches",
			selectors: []labels.Selector{us, eu},
			labels:    map[string]string{"region": "eu"},
			expected:  true,
		},
		{
			name:      "no selector matches",
			selectors: []labels.Selector{us, eu},
			labels:    map[string]string{"region": "ap"},
			expected:  false,
		},
		{
			name:      "nil labels",
			selectors: []labels.Selector{us},
			expected:  false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testCase.expected,
				AnySelectorMatches(testCase.selectors, testCase.labels),
			)
		})
	}
}

func TestFilterTargetsForStage(t *testing.T) {
	t.Parallel()

	newTarget := func(name string, lbls map[string]string) kargoapi.Target {
		return kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: name, Labels: lbls},
		}
	}
	newStage := func(selectors ...metav1.LabelSelector) *kargoapi.Stage {
		stage := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-stage"},
		}
		if selectors != nil {
			stage.Spec.Targets = &kargoapi.StageTargets{Selectors: selectors}
		}
		return stage
	}
	usEast := newTarget("us-east", map[string]string{"region": "us"})
	usWest := newTarget("us-west", map[string]string{"region": "us", "tier": "prod"})
	euWest := newTarget("eu-west", map[string]string{"region": "eu"})
	names := func(targets []kargoapi.Target) []string {
		out := make([]string, len(targets))
		for i, target := range targets {
			out[i] = target.Name
		}
		return out
	}

	testCases := []struct {
		name    string
		stage   *kargoapi.Stage
		targets []kargoapi.Target
		assert  func(*testing.T, []kargoapi.Target, error)
	}{
		{
			name:    "a classic Stage governs nothing",
			stage:   newStage(),
			targets: []kargoapi.Target{usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, targets)
			},
		},
		{
			name:    "selectors matching nothing yield an empty list, not nil",
			stage:   newStage(metav1.LabelSelector{MatchLabels: map[string]string{"region": "ap"}}),
			targets: []kargoapi.Target{usEast, euWest},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.NotNil(t, targets)
				require.Empty(t, targets)
			},
		},
		{
			name:    "matching Targets, sorted by name",
			stage:   newStage(metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}}),
			targets: []kargoapi.Target{usWest, euWest, usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"us-east", "us-west"}, names(targets))
			},
		},
		{
			name: "selectors describe a union",
			stage: newStage(
				metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}},
				metav1.LabelSelector{MatchLabels: map[string]string{"region": "eu"}},
			),
			targets: []kargoapi.Target{usEast, euWest},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"eu-west", "us-east"}, names(targets))
			},
		},
		{
			name: "a Target matching two selectors, or listed twice, appears once",
			stage: newStage(
				metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}},
				metav1.LabelSelector{MatchLabels: map[string]string{"tier": "prod"}},
			),
			targets: []kargoapi.Target{usWest, usWest},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"us-west"}, names(targets))
			},
		},
		{
			name:    "an empty selector selects everything",
			stage:   newStage(metav1.LabelSelector{}),
			targets: []kargoapi.Target{euWest, usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"eu-west", "us-east"}, names(targets))
			},
		},
		{
			name: "invalid selector",
			stage: newStage(metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "region",
					Operator: "NotAnOperator",
				}},
			}),
			targets: []kargoapi.Target{usEast},
			assert: func(t *testing.T, _ []kargoapi.Target, err error) {
				require.ErrorContains(t, err, "error parsing target selector 0")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			targets, err := FilterTargetsForStage(testCase.stage, testCase.targets)
			testCase.assert(t, targets, err)
		})
	}
}
