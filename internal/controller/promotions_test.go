package controller

import (
	"context"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/controller/runtime"
)

func TestNewPromotionReconciler(t *testing.T) {
	testConfig := config.ControllerConfig{
		BaseConfig: config.BaseConfig{
			LogLevel: log.DebugLevel,
		},
	}
	p := newPromotionReconciler(
		testConfig,
		fake.NewClientBuilder().Build(),
	)
	require.NotNil(t, p.client)
	require.NotNil(t, p.promoQueuesByEnv)
	require.NotNil(t, p.logger)
}

func TestInitializeQueues(t *testing.T) {
	scheme, err := api.SchemeBuilder.Build()
	require.NoError(t, err)
	reconciler := promotionReconciler{
		client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&api.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promotion",
					Namespace: "fake-namespace",
				},
				Spec: &api.PromotionSpec{
					Environment: "fake-environment",
				},
			},
		).Build(),
		promoQueuesByEnv: map[types.NamespacedName]runtime.PriorityQueue{},
		logger:           log.New(),
	}
	reconciler.logger.SetLevel(log.ErrorLevel)
	err = reconciler.initializeQueues(context.Background())
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
		err := pq.Push(&api.Promotion{
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
		promo := object.(*api.Promotion)
		if lastTime != nil {
			require.Greater(t, promo.CreationTimestamp.Time, *lastTime)
		}
		lastTime = &promo.CreationTimestamp.Time
	}
}

func TestPromotionSync(t *testing.T) {
	testCases := []struct {
		name       string
		promo      *api.Promotion
		pqs        map[types.NamespacedName]runtime.PriorityQueue
		assertions func(
			api.PromotionStatus,
			map[types.NamespacedName]runtime.PriorityQueue,
		)
	}{
		{
			// Existing promotions are listed at startup. We're only interested in
			// new ones. They're identifiable by lack of a phase.
			name: "existing promotion",
			promo: &api.Promotion{
				Status: api.PromotionStatus{
					Phase: api.PromotionPhasePending,
				},
			},
			assertions: func(
				status api.PromotionStatus,
				pqs map[types.NamespacedName]runtime.PriorityQueue,
			) {
				require.Equal(
					t,
					api.PromotionStatus{
						Phase: api.PromotionPhasePending, // Status should be unchanged
					},
					status,
				)
				require.Empty(t, pqs)
			},
		},

		{
			name: "promotion queue already exists",
			promo: &api.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
				Spec: &api.PromotionSpec{
					Environment: "fake-env",
				},
			},
			pqs: map[types.NamespacedName]runtime.PriorityQueue{
				{Namespace: "fake-namespace", Name: "fake-env"}: newPromotionsQueue(),
			},
			assertions: func(
				status api.PromotionStatus,
				pqs map[types.NamespacedName]runtime.PriorityQueue,
			) {
				require.Equal( // Status should have phase assigned
					t,
					api.PromotionStatus{
						Phase: api.PromotionPhasePending,
					},
					status,
				)
				pq, ok := pqs[types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-env",
				}]
				require.True(t, ok)
				require.Equal(t, 1, pq.Depth())
			},
		},

		{
			name: "promotion queue does not already exists",
			promo: &api.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
				Spec: &api.PromotionSpec{
					Environment: "fake-env",
				},
			},
			pqs: map[types.NamespacedName]runtime.PriorityQueue{},
			assertions: func(
				status api.PromotionStatus,
				pqs map[types.NamespacedName]runtime.PriorityQueue,
			) {
				require.Equal( // Status should have phase assigned
					t,
					api.PromotionStatus{
						Phase: api.PromotionPhasePending,
					},
					status,
				)
				pq, ok := pqs[types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-env",
				}]
				require.True(t, ok)
				require.Equal(t, 1, pq.Depth())
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := promotionReconciler{
				promoQueuesByEnv: testCase.pqs,
				logger:           log.New(),
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			testCase.assertions(
				reconciler.sync(context.Background(), testCase.promo),
				reconciler.promoQueuesByEnv,
			)
		})
	}
}

func TestSerializedSync(t *testing.T) {
	promo := &api.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-promo",
			Namespace: "fake-namespace",
		},
		Spec: &api.PromotionSpec{
			Environment: "fake-env",
			State:       "fake-state",
		},
	}

	scheme, err := api.SchemeBuilder.Build()
	require.NoError(t, err)
	client := fake.NewClientBuilder().
		WithScheme(scheme).WithObjects(promo).Build()

	pq := newPromotionsQueue()
	err = pq.Push(promo)
	require.NoError(t, err)

	reconciler := promotionReconciler{
		client: client,
		promoQueuesByEnv: map[types.NamespacedName]runtime.PriorityQueue{
			{Namespace: "fake-namespace", Name: "fake-env"}: pq,
		},
		logger: log.New(),
	}
	reconciler.logger.SetLevel(log.ErrorLevel)

	// Force the infinite loop under test to shut down after 3 seconds. This
	// should be plenty of time to handle the one Promotion we've given it.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reconciler.serializedSync(ctx, time.Second)

	// When we're done, the queue should be empty and the Promotion should be
	// complete.
	require.Equal(t, 0, pq.Depth())
	promo, err = reconciler.getPromo(
		ctx,
		types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promo",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, promo)
	require.Equal(t, api.PromotionPhaseComplete, promo.Status.Phase)
}

func TestGetPromo(t *testing.T) {
	scheme, err := api.SchemeBuilder.Build()
	require.NoError(t, err)

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*api.Promotion, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(promo *api.Promotion, err error) {
				require.NoError(t, err)
				require.Nil(t, promo)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&api.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promotion",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				},
			).Build(),
			assertions: func(promo *api.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-promotion", promo.Name)
				require.Equal(t, "fake-namespace", promo.Namespace)
				require.Equal(
					t,
					&api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
					promo.Spec,
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := promotionReconciler{
				client: testCase.client,
				logger: log.New(),
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			promo, err := reconciler.getPromo(
				context.Background(),
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
			)
			testCase.assertions(promo, err)
		})
	}
}
