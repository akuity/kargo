package promotions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewPromotionReconciler(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	r := newReconciler(
		client,
		client,
		&credentials.FakeDB{},
		bookkeeper.NewService(nil),
	)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.argoClient)
	require.NotNil(t, r.credentialsDB)
	require.NotNil(t, r.bookkeeperService)
	require.NotNil(t, r.promoQueuesByStage)

	// Assert that all overridable behaviors were initialized to a default:

	// Promotions (general):
	require.NotNil(t, r.promoteFn)
	require.NotNil(t, r.applyPromotionMechanismsFn)
	// Promotions via Git:
	require.NotNil(t, r.gitApplyUpdateFn)
	// Promotions via Git + Kustomize:
	require.NotNil(t, r.kustomizeSetImageFn)
	// Promotions via Git + Helm:
	require.NotNil(t, r.buildChartDependencyChangesFn)
	require.NotNil(t, r.updateChartDependenciesFn)
	require.NotNil(t, r.setStringsInYAMLFileFn)
	// Promotions via Argo CD:
	require.NotNil(t, r.getArgoCDAppFn)
	require.NotNil(t, r.applyArgoCDSourceUpdateFn)
	require.NotNil(t, r.argoCDAppPatchFn)
}

func TestInitializeQueues(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, api.SchemeBuilder.AddToScheme(scheme))
	r := reconciler{
		kargoClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&api.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promotion",
					Namespace: "fake-namespace",
				},
				Spec: &api.PromotionSpec{
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
		promo := object.(*api.Promotion) // nolint: forcetypeassert
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
					Stage: "fake-stage",
				},
			},
			pqs: map[types.NamespacedName]runtime.PriorityQueue{
				{Namespace: "fake-namespace", Name: "fake-stage"}: newPromotionsQueue(),
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
					Name:      "fake-stage",
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
					Stage: "fake-stage",
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
			status := r.sync(context.Background(), testCase.promo)
			testCase.assertions(
				status,
				r.promoQueuesByStage,
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
			Stage: "fake-stage",
			State: "fake-state",
		},
		Status: api.PromotionStatus{
			Phase: api.PromotionPhasePending,
		},
	}

	scheme := k8sruntime.NewScheme()
	require.NoError(t, api.SchemeBuilder.AddToScheme(scheme))
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
		promoteFn: func(context.Context, string, string, string) error {
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
	promo, err = r.getPromo(
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
	scheme := k8sruntime.NewScheme()
	require.NoError(t, api.SchemeBuilder.AddToScheme(scheme))

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
						Stage: "fake-stage",
						State: "fake-state",
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
						Stage: "fake-stage",
						State: "fake-state",
					},
					promo.Spec,
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := reconciler{
				kargoClient: testCase.client,
			}
			promo, err := r.getPromo(
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
