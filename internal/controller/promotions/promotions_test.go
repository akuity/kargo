package promotions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/directives"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
)

func TestNewPromotionReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	r := newReconciler(
		kubeClient,
		&fakeevent.EventRecorder{},
		&directives.FakeEngine{},
		ReconcilerConfig{},
	)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.recorder)
	require.NotNil(t, r.directivesEngine)
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
	return newReconciler(
		kargoClient,
		recorder,
		&directives.FakeEngine{},
		ReconcilerConfig{},
	)
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name      string
		promos    []client.Object
		promoteFn func(context.Context, v1alpha1.Promotion,
			*v1alpha1.Freight) (*kargoapi.PromotionStatus, error)
		terminateFn             func(context.Context, *kargoapi.Promotion) error
		promoToReconcile        *types.NamespacedName // if nil, uses the first of the promos
		expectPromoteFnCalled   bool
		expectTerminateFnCalled bool
		expectedPhase           kargoapi.PromotionPhase
		expectedEventRecorded   bool
		expectedEventReason     string
	}{
		{
			name:                  "normal reconcile",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseSucceeded,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionSucceeded,
			promos: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "fake-promo",
						},
					},
				},
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, now),
			},
			promoToReconcile: &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo"},
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
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "fake-promo",
						},
					},
				},
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhaseRunning, now),
			},
			promoToReconcile: &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo"},
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
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "fake-promo1",
						},
					},
				},
				newPromo("fake-namespace", "fake-promo1", "fake-stage", kargoapi.PromotionPhasePending, before),
				newPromo("fake-namespace", "fake-promo2", "fake-stage", kargoapi.PromotionPhasePending, now),
			},
		},
		{
			name:                  "stage not awaiting promo",
			expectPromoteFnCalled: false,
			promoToReconcile:      &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo"},
			expectedPhase:         kargoapi.PromotionPhaseRunning,
			expectedEventRecorded: false,
			promos: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "previous-promo",
						},
					},
				},
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, now),
			},
		},
		{
			name:                  "promoteFn panics",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseErrored,
			expectedEventRecorded: true,
			expectedEventReason:   kargoapi.EventReasonPromotionErrored,
			promos: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "fake-promo",
						},
					},
				},
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, before),
			},
			promoToReconcile: &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo"},
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
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "fake-promo",
						},
					},
				},
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhasePending, before),
			},
			promoToReconcile: &types.NamespacedName{Namespace: "fake-namespace", Name: "fake-promo"},
			promoteFn: func(_ context.Context, _ v1alpha1.Promotion, _ *v1alpha1.Freight) (*kargoapi.PromotionStatus, error) {
				return nil, errors.New("expected error")
			},
		},
		{
			name: "terminates promotion on request",
			promos: []client.Object{
				func() *kargoapi.Promotion {
					p := newPromo(
						"fake-namespace",
						"fake-promo",
						"fake-stage",
						kargoapi.PromotionPhasePending,
						now,
					)
					p.Annotations = map[string]string{
						kargoapi.AnnotationKeyAbort: string(kargoapi.AbortActionTerminate),
					}
					return p
				}(),
			},
			expectPromoteFnCalled:   false,
			expectTerminateFnCalled: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			recorder := fakeevent.NewEventRecorder(1)
			r := newFakeReconciler(t, recorder, tc.promos...)

			promoteWasCalled := false
			r.promoteFn = func(
				ctx context.Context,
				p v1alpha1.Promotion,
				_ *v1alpha1.Stage,
				f *v1alpha1.Freight,
			) (*kargoapi.PromotionStatus, error) {
				promoteWasCalled = true
				if tc.promoteFn != nil {
					return tc.promoteFn(ctx, p, f)
				}
				return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, nil
			}

			terminateWasCalled := false
			r.terminatePromotionFn = func(_ context.Context, promotion *kargoapi.Promotion) error {
				terminateWasCalled = true
				if tc.terminateFn != nil {
					return tc.terminateFn(ctx, promotion)
				}
				promotion.Status.Phase = kargoapi.PromotionPhaseAborted
				promotion.Status.Message = "terminated"
				promotion.Status.FinishedAt = &metav1.Time{Time: now.Time}
				return nil
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
			require.Equal(t, tc.expectTerminateFnCalled, terminateWasCalled,
				"terminateFn called: %t, expected %t", terminateWasCalled, tc.expectTerminateFnCalled)

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

func Test_reconciler_terminatePromotion(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	tests := []struct {
		name        string
		promo       *kargoapi.Promotion
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *kargoapi.Promotion, error)
	}{
		{
			name: "terminates pending promotion",
			promo: newPromo(
				"fake-namespace",
				"fake-promo",
				"fake-stage",
				kargoapi.PromotionPhasePending,
				now,
			),
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseAborted, promo.Status.Phase)
				require.Contains(t, promo.Status.Message, "terminated")
				require.NotNil(t, now, promo.Status.FinishedAt)
			},
		},
		{
			name: "promotion is already terminated",
			promo: func() *kargoapi.Promotion {
				p := newPromo(
					"fake-namespace",
					"fake-promo",
					"fake-stage",
					kargoapi.PromotionPhaseSucceeded,
					now,
				)
				p.Status.Message = "an existing message"
				return p
			}(),
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, promo.Status.Phase)
				require.Equal(t, "an existing message", promo.Status.Message)
			},
		},
		{
			name: "status patch error",
			promo: newPromo(
				"fake-namespace",
				"fake-promo",
				"fake-stage",
				kargoapi.PromotionPhasePending,
				now,
			),
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionPhasePending, promo.Status.Phase)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.promo).
				WithStatusSubresource(&kargoapi.Promotion{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &reconciler{
				kargoClient: c,
				recorder:    &fakeevent.EventRecorder{},
			}

			err := r.terminatePromotion(context.Background(), tt.promo)
			tt.assertions(t, tt.promo, err)
		})
	}
}

// Tests that initializeQueues is called properly
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
