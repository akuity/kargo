package applications

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

func TestIndexStagesByApp(t *testing.T) {
	const testShardName = "test-shard"
	testCases := []struct {
		name                string
		controllerShardName string
		stage               *kargoapi.Stage
		assertions          func([]string)
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
			assertions: func(res []string) {
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
			assertions: func(res []string) {
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
			assertions: func(res []string) {
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
			assertions: func(res []string) {
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			indexStagesByApp(testCase.controllerShardName)(testCase.stage)
		})
	}
}

func TestAppHealthChangePredicate(t *testing.T) {
	testCases := []struct {
		name    string
		old     map[string]any
		new     map[string]any
		updated bool
	}{
		{
			name: "health changed",
			old: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Degraded",
					},
				},
			},
			updated: true,
		},
		{
			name: "health did not change",
			old: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			updated: false,
		},
		{
			name: "sync status changed",
			old: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"health": map[string]any{
						"status": "Degraded",
					},
				},
			},
			updated: true,
		},
		{
			name: "sync status did not change",
			old: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"status": "Healthy",
					},
				},
			},
			new: map[string]any{
				"status": map[string]any{
					"sync": map[string]any{
						"status": "Healthy",
					},
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := AppHealthSyncStatusChangePredicate{}
			newUn := &unstructured.Unstructured{Object: testCase.new}
			oldUn := &unstructured.Unstructured{Object: testCase.old}
			updated := p.Update(event.UpdateEvent{
				ObjectNew: newUn,
				ObjectOld: oldUn,
			})
			require.Equal(t, testCase.updated, updated)
		})
	}
}
