package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetTarget(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.Target, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, target)
			},
		},
		{
			name: "error getting Target",
			client: fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return errors.New("something went wrong")
					},
				},
			).Build(),
			assertions: func(t *testing.T, target *kargoapi.Target, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error getting Target")
				require.Nil(t, target)
			},
		},
		{
			name: "success",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-target",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-target", target.Name)
				require.Equal(t, "fake-namespace", target.Namespace)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			target, err := GetTarget(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-target",
				},
			)
			testCase.assertions(t, target, err)
		})
	}
}

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

func TestFilterTargetsForStage(t *testing.T) {
	t.Parallel()
	newTarget := func(name string, lbls map[string]string) kargoapi.Target {
		return kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: name, Labels: lbls}}
	}
	usEast := newTarget("us-east", map[string]string{"region": "us"})
	usWest := newTarget("us-west", map[string]string{"region": "us"})
	euWest := newTarget("eu-west", map[string]string{"region": "eu"})
	both := newTarget("us-east", map[string]string{"region": "us", "tier": "prod"})
	targetAware := func(selectors ...metav1.LabelSelector) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "dev"},
			Spec:       kargoapi.StageSpec{Targets: &kargoapi.StageTargets{Selectors: selectors}},
		}
	}
	us := metav1.LabelSelector{MatchLabels: map[string]string{"region": "us"}}
	eu := metav1.LabelSelector{MatchLabels: map[string]string{"region": "eu"}}
	prod := metav1.LabelSelector{MatchLabels: map[string]string{"tier": "prod"}}

	testCases := []struct {
		name    string
		stage   *kargoapi.Stage
		targets []kargoapi.Target
		assert  func(*testing.T, []kargoapi.Target, error)
	}{
		{
			name:    "classic Stage governs nothing",
			stage:   &kargoapi.Stage{},
			targets: []kargoapi.Target{usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, targets)
			},
		},
		{
			name:    "target-aware Stage with no selectors governs nothing, but not nil",
			stage:   targetAware(),
			targets: []kargoapi.Target{usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.NotNil(t, targets)
				require.Empty(t, targets)
			},
		},
		{
			name: "malformed selector",
			stage: targetAware(metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "region", Operator: "Bogus"}},
			}),
			targets: []kargoapi.Target{usEast},
			assert: func(t *testing.T, _ []kargoapi.Target, err error) {
				require.ErrorContains(t, err, "error parsing target selector 0")
			},
		},
		{
			name:    "nothing matches",
			stage:   targetAware(us),
			targets: []kargoapi.Target{euWest},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.NotNil(t, targets)
				require.Empty(t, targets)
			},
		},
		{
			name:    "matches are sorted by name",
			stage:   targetAware(us),
			targets: []kargoapi.Target{usWest, euWest, usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.Target{usEast, usWest}, targets)
			},
		},
		{
			name:    "selectors describe a union",
			stage:   targetAware(us, eu),
			targets: []kargoapi.Target{usEast, euWest},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.Target{euWest, usEast}, targets)
			},
		},
		{
			name:    "a Target matching two selectors appears once",
			stage:   targetAware(us, prod),
			targets: []kargoapi.Target{both},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.Target{both}, targets)
			},
		},
		{
			name:    "a Target listed twice appears once",
			stage:   targetAware(us),
			targets: []kargoapi.Target{usEast, usEast},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.Target{usEast}, targets)
			},
		},
		{
			name:    "an empty selector selects everything",
			stage:   targetAware(metav1.LabelSelector{}),
			targets: []kargoapi.Target{usEast, euWest},
			assert: func(t *testing.T, targets []kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, []kargoapi.Target{euWest, usEast}, targets)
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
