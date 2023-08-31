package kubeclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

func TestIndexStagesByApp(t *testing.T) {
	const testShardName = "test-shard"
	t.Parallel()
	testCases := []struct {
		name                string
		controllerShardName string
		stage               *api.Stage
		assertions          func(*testing.T, []string)
	}{
		{
			name:                "Stage belongs to another shard",
			controllerShardName: testShardName,
			stage: &api.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controller.ShardLabelKey: "another-shard",
					},
				},
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCDAppUpdates: []api.ArgoCDAppUpdate{
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
			stage: &api.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controller.ShardLabelKey: testShardName,
					},
				},
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCDAppUpdates: []api.ArgoCDAppUpdate{
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
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCDAppUpdates: []api.ArgoCDAppUpdate{
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
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCDAppUpdates: []api.ArgoCDAppUpdate{
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
		input      *api.Promotion
		predicates []func(*api.Promotion) bool
		expected   []string
	}{
		"empty predicates/terminal phase": {
			input: &api.Promotion{
				Spec: &api.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhaseComplete,
				},
			},
			expected: []string{"fake-stage"},
		},
		"empty predicates/non-terminal phase": {
			input: &api.Promotion{
				Spec: &api.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhasePending,
				},
			},
			expected: []string{"fake-stage"},
		},
		"filter nonOutstandingPromotionPhase/terminal phase": {
			input: &api.Promotion{
				Spec: &api.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhaseComplete,
				},
			},
			predicates: []func(*api.Promotion) bool{
				filterNonOutstandingPromotionPhases,
			},
			expected: nil,
		},
		"filter nonOutstandingPromotionPhase/non-terminal phase": {
			input: &api.Promotion{
				Spec: &api.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhasePending,
				},
			},
			predicates: []func(*api.Promotion) bool{
				filterNonOutstandingPromotionPhases,
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
		policy     *api.PromotionPolicy
		assertions func(*testing.T, []string)
	}{
		{
			name: "promotion policy",
			policy: &api.PromotionPolicy{
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
