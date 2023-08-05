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

func TestIndexOutstandingPromotionsByStage(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		promotion  *api.Promotion
		assertions func(*testing.T, []string)
	}{
		{
			name: "promotion is in terminal phase",
			promotion: &api.Promotion{
				Spec: &api.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhaseComplete,
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name: "promotion is in non-terminal phase",
			promotion: &api.Promotion{
				Spec: &api.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: api.PromotionStatus{
					Phase: api.PromotionPhasePending,
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Equal(t, []string{"fake-stage"}, res)
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := indexOutstandingPromotionsByStage(tc.promotion)
			tc.assertions(t, res)
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
