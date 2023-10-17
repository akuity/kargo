package promotions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/bookkeeper"
	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewPromotionReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	r := newReconciler(
		kubeClient,
		kubeClient,
		&credentials.FakeDB{},
		bookkeeper.NewService(nil),
	)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.promoQueuesByStage)
	require.NotNil(t, r.promoteFn)
}

func TestInitializeQueues(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	r := reconciler{
		kargoClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promotion",
					Namespace: "fake-namespace",
				},
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
			},
		).Build(),
		promoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	err := r.initializeQueues(context.Background())
	require.NoError(t, err)
}

func TestNewPromotionsQueue(t *testing.T) {
	// runtime.PriorityQueue is already tested pretty well, so what we mainly
	// want to assert here is that our function for establishing relative priority
	// is correct.
	pq := newPromotionsQueue()

	// The last added should be the first out if our priority logic is correct
	now := time.Now()
	for i := 0; i < 100; i++ {
		err := pq.Push(&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(
					now.Add(-1 * time.Duration(i) * time.Minute),
				),
			},
		})
		require.NoError(t, err)
	}

	// Verify objects are prioritized by creation time
	var lastTime *time.Time
	for {
		object := pq.Pop()
		if object == nil {
			break
		}
		promo := object.(*kargoapi.Promotion) // nolint: forcetypeassert
		if lastTime != nil {
			require.Greater(t, promo.CreationTimestamp.Time, *lastTime)
		}
		lastTime = &promo.CreationTimestamp.Time
	}
}

func TestSyncPromo(t *testing.T) {
	testCases := []struct {
		name       string
		promo      *kargoapi.Promotion
		pqs        map[types.NamespacedName]runtime.PriorityQueue
		assertions func(
			kargoapi.PromotionStatus,
			map[types.NamespacedName]runtime.PriorityQueue,
		)
	}{
		{
			// Existing promotions are listed at startup. We're only interested in
			// new ones. They're identifiable by lack of a phase.
			name: "existing promotion",
			promo: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			assertions: func(
				status kargoapi.PromotionStatus,
				pqs map[types.NamespacedName]runtime.PriorityQueue,
			) {
				require.Equal(
					t,
					kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending, // Status should be unchanged
					},
					status,
				)
				require.Empty(t, pqs)
			},
		},

		{
			name: "promotion queue already exists",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
			},
			pqs: map[types.NamespacedName]runtime.PriorityQueue{
				{Namespace: "fake-namespace", Name: "fake-stage"}: newPromotionsQueue(),
			},
			assertions: func(
				status kargoapi.PromotionStatus,
				pqs map[types.NamespacedName]runtime.PriorityQueue,
			) {
				require.Equal( // Status should have phase assigned
					t,
					kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
					status,
				)
				pq, ok := pqs[types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-stage",
				}]
				require.True(t, ok)
				require.Equal(t, 1, pq.Depth())
			},
		},

		{
			name: "promotion queue does not already exists",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
				Spec: &kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
			},
			pqs: map[types.NamespacedName]runtime.PriorityQueue{},
			assertions: func(
				status kargoapi.PromotionStatus,
				pqs map[types.NamespacedName]runtime.PriorityQueue,
			) {
				require.Equal( // Status should have phase assigned
					t,
					kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
					status,
				)
				pq, ok := pqs[types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-stage",
				}]
				require.True(t, ok)
				require.Equal(t, 1, pq.Depth())
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := reconciler{
				promoQueuesByStage: testCase.pqs,
			}
			status := r.syncPromo(context.Background(), testCase.promo)
			testCase.assertions(
				status,
				r.promoQueuesByStage,
			)
		})
	}
}

func TestSerializedSync(t *testing.T) {
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-promo",
			Namespace: "fake-namespace",
		},
		Spec: &kargoapi.PromotionSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
		},
		Status: kargoapi.PromotionStatus{
			Phase: kargoapi.PromotionPhasePending,
		},
	}

	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	client := fake.NewClientBuilder().
		WithScheme(scheme).WithObjects(promo).Build()

	pq := newPromotionsQueue()
	err := pq.Push(promo)
	require.NoError(t, err)

	r := reconciler{
		kargoClient: client,
		promoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{
			{Namespace: "fake-namespace", Name: "fake-stage"}: pq,
		},
		promoteFn: func(context.Context, v1alpha1.Promotion) error {
			return nil
		},
	}

	// Force the infinite loop under test to shut down after 3 seconds. This
	// should be plenty of time to handle the one Promotion we've given it.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r.serializedSync(ctx, time.Second)

	// When we're done, the queue should be empty and the Promotion should be
	// complete.
	require.Equal(t, 0, pq.Depth())
	promo, err = kargoapi.GetPromotion(
		ctx,
		client,
		types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promo",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, promo)
	require.Equal(t, kargoapi.PromotionPhaseSucceeded, promo.Status.Phase)
}
