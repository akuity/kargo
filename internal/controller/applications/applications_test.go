package applications

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

func TestIndexStagesByApp(t *testing.T) {
	testCases := []struct {
		name            string
		controllerShard string
		stage           *api.Stage
		assertions      func([]string)
	}{
		{
			name: "stage has no health checks",
			stage: &api.Stage{
				Spec: &api.StageSpec{},
			},
			assertions: func(res []string) {
				require.Nil(t, res)
			},
		},

		{
			name:            "stage has health checks, but belongs to another shard",
			controllerShard: "foo",
			stage: &api.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controller.ShardLabelKey: "bar",
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
			assertions: func(res []string) {
				require.Nil(t, res)
			},
		},

		{
			name: "stage has health checks",
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						ArgoCDAppUpdates: []api.ArgoCDAppUpdate{
							{
								AppNamespace: "fake-namespace",
								AppName:      "fake-app",
							},
							{
								AppNamespace: "another-fake-namespace",
								AppName:      "another-fake-app",
							},
						},
					},
				},
			},
			assertions: func(res []string) {
				require.Equal(
					t,
					[]string{
						"fake-namespace:fake-app",
						"another-fake-namespace:another-fake-app",
					},
					res,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			indexStagesByApp(testCase.controllerShard)(testCase.stage)
		})
	}
}
