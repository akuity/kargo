package promotions

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
	"github.com/akuity/kargo/internal/promotion"
	pkgPromotion "github.com/akuity/kargo/pkg/promotion"
)

var (
	now    = metav1.Now()
	before = metav1.Time{Time: now.Add(time.Second * -1)}
)

func TestNewPromotionReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	r := newReconciler(
		kubeClient,
		&fakeevent.EventRecorder{},
		&promotion.MockEngine{},
		ReconcilerConfig{},
	)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.recorder)
	require.NotNil(t, r.promoEngine)
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
		&promotion.MockEngine{},
		ReconcilerConfig{},
	)
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name      string
		promos    []client.Object
		promoteFn func(context.Context, kargoapi.Promotion,
			*kargoapi.Freight) (*kargoapi.PromotionStatus, error)
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
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-namespace",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "other-fake-promo1",
						},
					},
				},
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
			expectedPhase:         kargoapi.PromotionPhasePending,
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
			promoteFn: func(_ context.Context, _ kargoapi.Promotion, _ *kargoapi.Freight) (*kargoapi.PromotionStatus, error) {
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
			promoteFn: func(_ context.Context, _ kargoapi.Promotion, _ *kargoapi.Freight) (*kargoapi.PromotionStatus, error) {
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
				p kargoapi.Promotion,
				_ *kargoapi.Stage,
				f *kargoapi.Freight,
			) (*kargoapi.PromotionStatus, error) {
				promoteWasCalled = true
				if tc.promoteFn != nil {
					return tc.promoteFn(ctx, p, f)
				}
				return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, nil
			}

			terminateWasCalled := false
			r.terminatePromotionFn = func(
				_ context.Context,
				_ *kargoapi.AbortPromotionRequest,
				promotion *kargoapi.Promotion,
				_ *kargoapi.Freight,
			) error {
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
		req         kargoapi.AbortPromotionRequest
		promo       *kargoapi.Promotion
		freight     *kargoapi.Freight
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *fakeevent.EventRecorder, *kargoapi.Promotion, error)
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
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseAborted, promo.Status.Phase)
				require.Contains(t, promo.Status.Message, "terminated")
				require.NotNil(t, promo.Status.FinishedAt)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionAborted, event.Reason)
			},
		},
		{
			name: "terminates running promotion",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: now,
					Name:              "fake-promo",
					Namespace:         "fake-namespace",
				},
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase:       kargoapi.PromotionPhaseRunning,
					CurrentStep: 0,
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						StartedAt: &now,
						Status:    kargoapi.PromotionStepStatusRunning,
					}},
				},
			},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseAborted, promo.Status.Phase)
				require.Contains(t, promo.Status.Message, "terminated")
				require.NotNil(t, promo.Status.FinishedAt)

				require.Equal(t, kargoapi.PromotionStepStatusAborted, promo.Status.StepExecutionMetadata[0].Status)
				require.NotNil(t, promo.Status.StepExecutionMetadata[0].FinishedAt)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionAborted, event.Reason)
			},
		},
		{
			name: "emits event with actor",
			req: kargoapi.AbortPromotionRequest{
				Actor: "fake-actor",
			},
			promo: newPromo(
				"fake-namespace",
				"fake-promo",
				"fake-stage",
				kargoapi.PromotionPhasePending,
				now,
			),
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseAborted, promo.Status.Phase)
				require.Contains(t, promo.Status.Message, "terminated")
				require.NotNil(t, promo.Status.FinishedAt)

				require.Len(t, recorder.Events, 1)
				event := <-recorder.Events
				require.Equal(t, kargoapi.EventReasonPromotionAborted, event.Reason)
				actor := event.Annotations[kargoapi.AnnotationKeyEventActor]
				require.Equal(t, "fake-actor", actor)
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
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionPhaseSucceeded, promo.Status.Phase)
				require.Equal(t, "an existing message", promo.Status.Message)
				require.Len(t, recorder.Events, 0)
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
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder, promo *kargoapi.Promotion, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionPhasePending, promo.Status.Phase)
				require.Len(t, recorder.Events, 0)
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
			recorder := fakeevent.NewEventRecorder(1)

			r := &reconciler{
				kargoClient: c,
				recorder:    recorder,
			}

			req := tt.req
			err := r.terminatePromotion(context.Background(), &req, tt.promo, tt.freight)
			tt.assertions(t, recorder, tt.promo, err)
		})
	}
}

func Test_parseCreateActorAnnotation(t *testing.T) {
	tests := []struct {
		name  string
		promo *kargoapi.Promotion
		want  string
	}{
		{
			name: "basic case",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promo",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: fmt.Sprintf(
							"%s%s", kargoapi.EventActorEmailPrefix, "fake-actor",
						),
					},
				},
			},
			want: "fake-actor",
		},
		{
			name: "single element",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promo",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: kargoapi.EventActorAdmin,
					},
				},
			},
			want: kargoapi.EventActorAdmin,
		},
		{
			name: "unknown actor",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promo",
					Namespace: "fake-namespace",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyCreateActor: kargoapi.EventActorUnknown,
					},
				},
			},
			want: "",
		},
		{

			name: "no annotation",
			promo: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-promo",
					Namespace: "fake-namespace",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCreateActorAnnotation(tt.promo)
			require.Equal(t, tt.want, result)
		})
	}
}

func Test_calculateRequeueInterval(t *testing.T) {
	testStepKindWithoutTimeout := "fake-step-without-timeout"
	promotion.RegisterStepRunner(
		&pkgPromotion.MockStepRunner{Nm: testStepKindWithoutTimeout},
	)

	testStepKindWithTimeout := "fake-step-with-timeout"
	testTimeout := 10 * time.Minute
	promotion.RegisterStepRunner(
		pkgPromotion.NewRetryableStepRunner(
			&pkgPromotion.MockStepRunner{Nm: testStepKindWithTimeout},
			&testTimeout,
			0, // Retries don't matter for this test
		),
	)

	// The test cases are crafted with the assumption that the default requeue
	// interval is greater than one minute, so we need to assert that this is the
	// case.
	require.Greater(t, defaultRequeueInterval, time.Minute)

	// The test cases are crafted with the assumption that the step's timeout is
	// greater than the default requeue interval, so we need to assert that this
	// is the case.
	require.Greater(t, testTimeout, defaultRequeueInterval)

	testCases := []struct {
		name       string
		promo      *kargoapi.Promotion
		assertions func(*testing.T, time.Duration)
	}{
		{
			name: "no timeout",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses: testStepKindWithoutTimeout,
					}},
				},
				Status: kargoapi.PromotionStatus{
					CurrentStep: 0,
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						StartedAt: &metav1.Time{Time: time.Now()},
					}},
				},
			},
			assertions: func(t *testing.T, requeueInterval time.Duration) {
				// The request should be requeued according to the default.
				require.Equal(t, defaultRequeueInterval, requeueInterval)
			},
		},
		{
			name: "timeout occurs after next interval",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses: testStepKindWithTimeout,
					}},
				},
				Status: kargoapi.PromotionStatus{
					CurrentStep: 0,
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						// If the step started now and times out after an interval greater
						// than the default requeue interval, then the wall clock time of
						// the timeout will be AFTER the wall clock time of the next
						// reconciliation.
						StartedAt: &metav1.Time{Time: time.Now()},
					}},
				},
			},
			assertions: func(t *testing.T, requeueInterval time.Duration) {
				// The request should be requeued according to the default.
				require.Equal(t, defaultRequeueInterval, requeueInterval)
				// Sanity check that the requeue interval is always greater than 0.
				require.Greater(t, requeueInterval, time.Duration(0))
			},
		},
		{
			name: "timeout occurs before next interval",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses: testStepKindWithTimeout,
					}},
				},
				Status: kargoapi.PromotionStatus{
					CurrentStep: 0,
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						// If the step has only a minute to go before timeout, then the wall
						// clock time of the timeout will be BEFORE the wall clock time of
						// the next reconciliation.
						StartedAt: &metav1.Time{
							Time: metav1.Now().Add(-testTimeout).Add(time.Minute),
						},
					}},
				},
			},
			assertions: func(t *testing.T, requeueInterval time.Duration) {
				// The interval to the next reconciliation should be shortened.
				require.Less(t, requeueInterval, defaultRequeueInterval)
				// Sanity check that the requeue interval is always greater than 0.
				require.Greater(t, requeueInterval, time.Duration(0))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, calculateRequeueInterval(testCase.promo))
		})
	}
}

// nolint: unparam
func newPromo(namespace, name, stage string,
	phase kargoapi.PromotionPhase,
	creationTimestamp metav1.Time,
) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			Name:              name,
			Namespace:         namespace,
		},
		Spec: kargoapi.PromotionSpec{
			Stage: stage,
		},
		Status: kargoapi.PromotionStatus{
			Phase: phase,
		},
	}
}
