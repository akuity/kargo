package kubeclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

func TestIndexStagesByApp(t *testing.T) {
	const testShardName = "test-shard"
	t.Parallel()
	testCases := []struct {
		name                string
		controllerShardName string
		stage               *kargoapi.Stage
		assertions          func(*testing.T, []string)
	}{
		{
			name:                "Stage belongs to another shard",
			controllerShardName: testShardName,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controller.ShardLabelKey: "another-shard",
					},
				},
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-app",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name:                "Stage belongs to this shard",
			controllerShardName: testShardName,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controller.ShardLabelKey: testShardName,
					},
				},
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-app",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Equal(
					t,
					[]string{
						"fake-namespace:fake-app",
					},
					res,
				)
			},
		},
		{
			name:                "Stage is unlabeled and this is not the default controller",
			controllerShardName: testShardName,
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-app",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name:                "Stage is unlabeled and this is the default controller",
			controllerShardName: "",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-app",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Equal(
					t,
					[]string{
						"fake-namespace:fake-app",
					},
					res,
				)
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := indexStagesByArgoCDApplications(tc.controllerShardName)(tc.stage)
			tc.assertions(t, res)
		})
	}
}

func TestIndexPromotionsByStage(t *testing.T) {
	testCases := map[string]struct {
		input      *kargoapi.Promotion
		predicates []func(*kargoapi.Promotion) bool
		expected   []string
	}{
		"empty predicates/terminal phase": {
			input: &kargoapi.Promotion{
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			expected: []string{"fake-stage"},
		},
		"empty predicates/non-terminal phase": {
			input: &kargoapi.Promotion{
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			expected: []string{"fake-stage"},
		},
		"isPromotionPhaseNonTerminal excludes Promotions in terminal phases": {
			input: &kargoapi.Promotion{
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			predicates: []func(*kargoapi.Promotion) bool{
				isPromotionPhaseNonTerminal,
			},
			expected: nil,
		},
		"isPromotionPhaseNonTerminal selects Promotions in non-terminal phases": {
			input: &kargoapi.Promotion{
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			predicates: []func(*kargoapi.Promotion) bool{
				isPromotionPhaseNonTerminal,
			},
			expected: []string{"fake-stage"},
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			actual := indexPromotionsByStage(tc.predicates...)(tc.input)
			require.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func TestIndexPromotionPoliciesByStage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		policy     *kargoapi.PromotionPolicy
		assertions func(*testing.T, []string)
	}{
		{
			name: "promotion policy",
			policy: &kargoapi.PromotionPolicy{
				Stage: "fake-stage",
			},
			assertions: func(t *testing.T, res []string) {
				require.Equal(t, []string{"fake-stage"}, res)
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := indexPromotionPoliciesByStage(tc.policy)
			tc.assertions(t, res)
		})
	}
}
