package stages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func TestEnqueueDownstreamStagesHandler(t *testing.T) {
	downstreamStage := v1alpha1.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "downstream",
			Namespace: "kargo-test",
		},
		Spec: &v1alpha1.StageSpec{
			Subscriptions: &v1alpha1.Subscriptions{
				UpstreamStages: []v1alpha1.StageSubscription{
					{
						Name: "upstream",
					},
				},
			},
		},
	}
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	testCases := []struct {
		name      string
		updateEvt *event.UpdateEvent
		stages    []*v1alpha1.Stage
		enqueued  bool
	}{
		{
			name: "old is nil",
			updateEvt: &event.UpdateEvent{
				ObjectOld: nil,
				ObjectNew: &v1alpha1.Stage{},
			},
			enqueued: false,
		},
		{
			name: "new is nil",
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Stage{},
				ObjectNew: nil,
			},
			enqueued: false,
		},
		{
			name: "new item in history",
			stages: []*v1alpha1.Stage{
				&downstreamStage,
			},
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Stage{},
				ObjectNew: &v1alpha1.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "upstream",
						Namespace: "kargo-test",
					},
					Spec: &v1alpha1.StageSpec{
						Subscriptions: &v1alpha1.Subscriptions{
							UpstreamStages: []v1alpha1.StageSubscription{
								{
									Name: "downstream",
								},
							},
						},
					},
					Status: v1alpha1.StageStatus{
						History: []v1alpha1.Freight{
							{
								ID: "abc123",
							},
						},
					},
				},
			},
			enqueued: true,
		},
		{
			name: "history changed",
			stages: []*v1alpha1.Stage{
				&downstreamStage,
			},
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "upstream",
						Namespace: "kargo-test",
					},
					Spec: &v1alpha1.StageSpec{
						Subscriptions: &v1alpha1.Subscriptions{
							UpstreamStages: []v1alpha1.StageSubscription{
								{
									Name: "downstream",
								},
							},
						},
					},
					Status: v1alpha1.StageStatus{
						History: []v1alpha1.Freight{
							{
								ID: "abc123",
							},
						},
					},
				},
				ObjectNew: &v1alpha1.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "upstream",
						Namespace: "kargo-test",
					},
					Spec: &v1alpha1.StageSpec{
						Subscriptions: &v1alpha1.Subscriptions{
							UpstreamStages: []v1alpha1.StageSubscription{
								{
									Name: "downstream",
								},
							},
						},
					},
					Status: v1alpha1.StageStatus{
						History: []v1alpha1.Freight{
							{
								ID: "def456",
							},
						},
					},
				},
			},
			enqueued: true,
		},
		{
			name: "history did not change",
			stages: []*v1alpha1.Stage{
				&downstreamStage,
			},
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "upstream",
						Namespace: "kargo-test",
					},
					Spec: &v1alpha1.StageSpec{
						Subscriptions: &v1alpha1.Subscriptions{
							UpstreamStages: []v1alpha1.StageSubscription{
								{
									Name: "downstream",
								},
							},
						},
					},
					Status: v1alpha1.StageStatus{
						History: []v1alpha1.Freight{
							{
								ID: "abc123",
							},
						},
					},
				},
				ObjectNew: &v1alpha1.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "upstream",
						Namespace: "kargo-test",
					},
					Spec: &v1alpha1.StageSpec{
						Subscriptions: &v1alpha1.Subscriptions{
							UpstreamStages: []v1alpha1.StageSubscription{
								{
									Name: "downstream",
								},
							},
						},
					},
					Status: v1alpha1.StageStatus{
						History: []v1alpha1.Freight{
							{
								ID: "abc123",
							},
						},
					},
				},
			},
			enqueued: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var objs []client.Object
			for _, stage := range testCase.stages {
				objs = append(objs, stage)
			}
			hand := EnqueueDownstreamStagesHandler{
				logger:      logging.LoggerFromContext(context.TODO()),
				kargoClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
			}
			wq := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			if testCase.updateEvt != nil {
				hand.Update(*testCase.updateEvt, wq)
				added := wq.Len() > 0
				require.Equal(
					t,
					testCase.enqueued,
					added,
				)
			}
		})
	}

}

func TestPromoWentTerminal(t *testing.T) {
	testCases := []struct {
		name      string
		updateEvt *event.UpdateEvent
		deleteEvt *event.DeleteEvent
		expected  bool
	}{
		{
			name: "went terminal",
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseRunning,
					},
				},
				ObjectNew: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseSucceeded,
					},
				},
			},
			expected: true,
		},
		{
			name: "stayed running",
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseRunning,
					},
				},
				ObjectNew: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseRunning,
					},
				},
			},
			expected: false,
		},
		{
			name: "stayed terminal",
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseSucceeded,
					},
				},
				ObjectNew: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseSucceeded,
					},
				},
			},
			expected: false,
		},
		{
			name: "new is nil",
			updateEvt: &event.UpdateEvent{
				ObjectOld: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseSucceeded,
					},
				},
				ObjectNew: nil,
			},
			expected: false,
		},
		{
			name: "old is nil",
			updateEvt: &event.UpdateEvent{
				ObjectOld: nil,
				ObjectNew: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseSucceeded,
					},
				},
			},
			expected: false,
		},
		{
			name: "deleted already terminal",
			deleteEvt: &event.DeleteEvent{
				Object: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseSucceeded,
					},
				},
			},
			expected: false,
		},
		{
			name: "deleted non-terminal",
			deleteEvt: &event.DeleteEvent{
				Object: &v1alpha1.Promotion{
					Status: v1alpha1.PromotionStatus{
						Phase: v1alpha1.PromotionPhaseRunning,
					},
				},
			},
			expected: true,
		},
		{
			name: "deleted object is nil",
			deleteEvt: &event.DeleteEvent{
				Object: nil,
			},
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pt := PromoWentTerminal{logger: logging.LoggerFromContext(context.TODO())}
			if testCase.updateEvt != nil {
				require.Equal(
					t,
					testCase.expected,
					pt.Update(*testCase.updateEvt),
				)
			}
			if testCase.deleteEvt != nil {
				require.Equal(
					t,
					testCase.expected,
					pt.Delete(*testCase.deleteEvt),
				)
			}
		})
	}
}
