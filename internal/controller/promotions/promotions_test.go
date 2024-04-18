package promotions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
)

func TestNewPromotionReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	r := newReconciler(
		kubeClient,
		kubeClient,
		&fakeevent.EventRecorder{},
		&credentials.FakeDB{},
		ReconcilerConfig{},
	)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.pqs.pendingPromoQueuesByStage)
	require.NotNil(t, r.getStageFn)
	require.NotNil(t, r.promoteFn)
}

func newFakeReconciler(
	t *testing.T,
	recorder *fakeevent.EventRecorder,
	objects ...client.Object,
) *reconciler {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	kargoClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objects...).WithStatusSubresource(objects...).Build()
	kubeClient := fake.NewClientBuilder().Build()
	return newReconciler(
		kargoClient,
		kubeClient,
		recorder,
		&credentials.FakeDB{},
		ReconcilerConfig{},
	)
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name      string
		promos    []client.Object
		promoteFn func(context.Context, v1alpha1.Promotion,
			*v1alpha1.Freight) (*kargoapi.PromotionStatus, error)
		promoToReconcile      *types.NamespacedName // if nil, uses the first of the promos
		expectPromoteFnCalled bool
		expectedPhase         kargoapi.PromotionPhase
		expectedEventRecorded bool
		expectedEventReason   string
	}{
		{
			name:                  "normal reconcile",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseSucceeded,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionSucceeded,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, now),
			},
		},
		{
			name:                  "promo doesn't exist",
			promoToReconcile:      &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo"},
			expectPromoteFnCalled: false,
		},
		{
			name:                  "promo already completed",
			expectPromoteFnCalled: false,
			expectedPhase:         kargoapi.PromotionPhaseErrored,
			expectedEventRecorded: false,
			expectedEventReason:   kargoapi.EventReasonPromotionErrored,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhaseErrored, now),
			},
		},
		{
			name:                  "promo already running",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseSucceeded,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionSucceeded,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhaseRunning, now),
			},
		},
		{
			name:                  "promo does not have highest priority",
			expectPromoteFnCalled: false,
			promoToReconcile:      &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo2"},
			expectedPhase:         kargoapi.PromotionPhasePending,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo1", "fake-stage", kargoapi.PromotionPhasePending, before),
				newPromo("fake-namespace", "fake-promo2", "fake-stage", "", now), // intentionally empty string phase
			},
		},
		{
			name:                  "promo has highest priority",
			expectPromoteFnCalled: true,
			promoToReconcile:      &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo1"},
			expectedPhase:         kargoapi.PromotionPhaseSucceeded,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionSucceeded,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo1", "fake-stage", kargoapi.PromotionPhasePending, before),
				newPromo("fake-namespace", "fake-promo2", "fake-stage", kargoapi.PromotionPhasePending, now),
			},
		},
		{
			name:                  "promoteFn panics",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseErrored,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionErrored,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, before),
			},
			promoteFn: func(_ context.Context, _ v1alpha1.Promotion, _ *v1alpha1.Freight) (*kargoapi.PromotionStatus, error) {
				panic("expected panic")
			},
		},
		{
			name:                  "promoteFn errors",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseErrored,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionErrored,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, before),
			},
			promoteFn: func(_ context.Context, _ v1alpha1.Promotion, _ *v1alpha1.Freight) (*kargoapi.PromotionStatus, error) {
				return nil, errors.New("expected error")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			recorder := fakeevent.NewEventRecorder(1)
			r := newFakeReconciler(t, recorder, tc.promos...)
			promoteWasCalled := false
			r.getStageFn = func(context.Context, client.Client, types.NamespacedName) (*kargoapi.Stage, error) {
				return &kargoapi.Stage{
					Spec: &kargoapi.StageSpec{},
				}, nil
			}
			r.promoteFn = func(ctx context.Context, p v1alpha1.Promotion,
				f *v1alpha1.Freight) (*kargoapi.PromotionStatus, error) {
				promoteWasCalled = true
				if tc.promoteFn != nil {
					return tc.promoteFn(ctx, p, f)
				}
				return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, nil
			}
			var req ctrl.Request
			if tc.promoToReconcile != nil {
				req = ctrl.Request{NamespacedName: *tc.promoToReconcile}
			} else {
				req = ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: tc.promos[0].GetNamespace(),
					Name:      tc.promos[0].GetName(),
				}}
			}

			_, err := r.Reconcile(ctx, req)
			require.NoError(t, err)
			require.Equal(t, tc.expectPromoteFnCalled, promoteWasCalled,
				"promoteFn called: %t, expected %t", promoteWasCalled, tc.expectPromoteFnCalled)

			if tc.expectedPhase != "" {
				var updatedPromo kargoapi.Promotion
				err = r.kargoClient.Get(ctx, req.NamespacedName, &updatedPromo)
				require.NoError(t, err)
				require.Equal(t, tc.expectedPhase, updatedPromo.Status.Phase)
				if tc.expectedEventRecorded {
					require.Len(t, recorder.Events, 1)
					event := <-recorder.Events
					require.Equal(t, tc.expectedEventReason, event.Reason)
				}
			}
		})
	}
}

// Tests that initalizeQueues is called properly
func TestReconcileInitializeQueues(t *testing.T) {
	ctx := context.TODO()
	promos := []client.Object{
		newPromo("fake-namespace", "fake-promo1", "fake-stage", kargoapi.PromotionPhasePending, before),
		newPromo("fake-namespace", "fake-promo2", "fake-stage", kargoapi.PromotionPhasePending, now),
	}
	recorder := &fakeevent.EventRecorder{}
	r := newFakeReconciler(t, recorder, promos...)

	// reconcile a non-existent promo to trigger initializeQueues
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "does-not-exist", Name: "does-not-exist"}}

	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)

	// Verifies queues got set up
	stageKey := types.NamespacedName{Namespace: "fake-namespace", Name: "fake-stage"}
	require.Equal(t, 2, r.pqs.pendingPromoQueuesByStage[stageKey].Depth())
}
