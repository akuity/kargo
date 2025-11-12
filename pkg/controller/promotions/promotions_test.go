package promotions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/promotion"
)

var (
	now    = metav1.Now()
	before = metav1.Time{Time: now.Add(time.Second * -1)}
)

func TestNewPromotionReconciler(t *testing.T) {
	kubeClient := fake.NewClientBuilder().Build()
	r := newReconciler(
		kubeClient,
		k8sevent.NewEventSender(&fakeevent.EventRecorder{}),
		&promotion.MockEngine{},
		ReconcilerConfig{},
	)
	require.NotNil(t, r.kargoClient)
	require.NotNil(t, r.sender)
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
		k8sevent.NewEventSender(recorder),
		&promotion.MockEngine{},
		ReconcilerConfig{},
	)
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name      string
		promos    []client.Object
		promoteFn func(
			context.Context,
			kargoapi.Promotion,
			*kargoapi.Freight,
		) (*kargoapi.PromotionStatus, *time.Duration, error)
		terminateFn             func(context.Context, *kargoapi.Promotion) error
		promoToReconcile        *types.NamespacedName // if nil, uses the first of the promos
		expectPromoteFnCalled   bool
		expectTerminateFnCalled bool
		expectedPhase           kargoapi.PromotionPhase
		expectedEventRecorded   bool
		expectedEventType       kargoapi.EventType
	}{
		{
			name:                  "normal reconcile",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseSucceeded,
			expectedEventRecorded: true,
			expectedEventType:     kargoapi.EventTypePromotionSucceeded,
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
			expectedEventType:     kargoapi.EventTypePromotionErrored,
			promos: []client.Object{
				newPromo("fake-namespace", "fake-promo", "fake-stage", kargoapi.PromotionPhaseErrored, now),
			},
		},
		{
			name:                  "Promotion doesn't belong to shard",
			expectPromoteFnCalled: false,
			expectedPhase:         kargoapi.PromotionPhasePending,
			expectedEventRecorded: false,
			promos: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promo",
						Namespace: "fake-namespace",
						Labels: map[string]string{
							kargoapi.LabelKeyShard: "wrong-shard",
						},
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
		},
		{
			name:                  "promo already running",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseSucceeded,
			expectedEventRecorded: true,
			expectedEventType:     kargoapi.EventTypePromotionSucceeded,
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
			expectedEventType:     kargoapi.EventTypePromotionSucceeded,
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
			expectedEventType:     kargoapi.EventTypePromotionErrored,
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
			promoteFn: func(
				context.Context,
				kargoapi.Promotion,
				*kargoapi.Freight,
			) (*kargoapi.PromotionStatus, *time.Duration, error) {
				panic("expected panic")
			},
		},
		{
			name:                  "promoteFn errors",
			expectPromoteFnCalled: true,
			expectedPhase:         kargoapi.PromotionPhaseErrored,
			expectedEventRecorded: true,
			expectedEventType:     kargoapi.EventTypePromotionErrored,
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
			promoteFn: func(
				context.Context,
				kargoapi.Promotion,
				*kargoapi.Freight,
			) (*kargoapi.PromotionStatus, *time.Duration, error) {
				return nil, nil, errors.New("expected error")
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
			) (*kargoapi.PromotionStatus, *time.Duration, error) {
				promoteWasCalled = true
				if tc.promoteFn != nil {
					return tc.promoteFn(ctx, p, f)
				}
				return &kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				}, nil, nil
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
					require.Equal(t, tc.expectedEventType, kargoapi.EventType(event.Reason))
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
				require.Equal(t, string(kargoapi.EventTypePromotionAborted), event.Reason)
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
				require.Equal(t, string(kargoapi.EventTypePromotionAborted), event.Reason)
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
				require.Equal(t, string(kargoapi.EventTypePromotionAborted), event.Reason)
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
				sender:      k8sevent.NewEventSender(recorder),
			}

			req := tt.req
			err := r.terminatePromotion(context.Background(), &req, tt.promo, tt.freight)
			tt.assertions(t, recorder, tt.promo, err)
		})
	}
}

func Test_calculateRequeueInterval(t *testing.T) {
	testStepKindWithoutTimeout := "fake-step-without-timeout"
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: testStepKindWithoutTimeout,
			Value: func(promotion.StepRunnerCapabilities) promotion.StepRunner {
				return &promotion.MockStepRunner{}
			},
		},
	)

	testStepKindWithTimeout := "fake-step-with-timeout"
	testTimeout := 10 * time.Minute
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:     testStepKindWithTimeout,
			Metadata: promotion.StepRunnerMetadata{DefaultTimeout: testTimeout},
			Value: func(promotion.StepRunnerCapabilities) promotion.StepRunner {
				return &promotion.MockStepRunner{}
			},
		},
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
		name                     string
		promo                    *kargoapi.Promotion
		suggestedRequeueInterval *time.Duration
		assertions               func(*testing.T, time.Duration)
	}{
		{
			name: "current step out of bounds",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses: testStepKindWithoutTimeout,
					}},
				},
				Status: kargoapi.PromotionStatus{
					CurrentStep: 1,
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						StartedAt: &metav1.Time{Time: time.Now()},
					}},
				},
			},
			assertions: func(t *testing.T, requeueInterval time.Duration) {
				require.Equal(t, defaultRequeueInterval, requeueInterval)
			},
		},
		{
			name: "nil step execution metadata",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses: testStepKindWithoutTimeout,
					}},
				},
				Status: kargoapi.PromotionStatus{
					CurrentStep:           0,
					StepExecutionMetadata: nil,
				},
			},
			assertions: func(t *testing.T, requeueInterval time.Duration) {
				require.Equal(t, defaultRequeueInterval, requeueInterval)
			},
		},
		{
			name: "step execution metadata out of bounds",
			promo: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{{
						Uses: testStepKindWithoutTimeout,
					}},
				},
				Status: kargoapi.PromotionStatus{
					CurrentStep:           0,
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{},
				},
			},
			assertions: func(t *testing.T, requeueInterval time.Duration) {
				require.Equal(t, defaultRequeueInterval, requeueInterval)
			},
		},
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
			name:                     "timeout would occur after suggested interval elapses",
			suggestedRequeueInterval: ptr.To(time.Minute),
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
				// The request should be requeued according to the suggestion.
				require.Equal(t, time.Minute, requeueInterval)
				// Sanity check that the requeue interval is always greater than 0.
				require.Greater(t, requeueInterval, time.Duration(0))
			},
		},
		{
			name:                     "timeout would occur before suggested interval elapses",
			suggestedRequeueInterval: ptr.To(time.Minute),
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
				require.Less(t, requeueInterval, time.Minute)
				// Sanity check that the requeue interval is always greater than 0.
				require.Greater(t, requeueInterval, time.Duration(0))
			},
		},
		{
			name: "timeout would occur after default interval elapses",
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
			name: "timeout would occur before next default interval elapses",
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
			testCase.assertions(
				t,
				calculateRequeueInterval(
					t.Context(),
					testCase.promo,
					testCase.suggestedRequeueInterval,
				),
			)
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

func Test_buildTargetFreightCollection(t *testing.T) {
	testCases := []struct {
		name                      string
		targetFreight             kargoapi.FreightReference
		stage                     *kargoapi.Stage
		expectedNumFreight        int
		expectedFreightCollection *kargoapi.FreightCollection
	}{
		{
			name:          "requested freight not greater than 1",
			targetFreight: kargoapi.FreightReference{Name: "target-freight"},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}},
				},
			},
			expectedNumFreight: 1,
		},
		{
			name:          "no last promotion should not panic",
			targetFreight: kargoapi.FreightReference{Name: "target-freight"},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}, {}},
				},
				Status: kargoapi.StageStatus{LastPromotion: nil},
			},
			expectedNumFreight: 1,
		},
		{
			name:          "no last promotion status should not panic",
			targetFreight: kargoapi.FreightReference{Name: "target-freight"},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}, {}},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{Status: nil},
				},
			},
			expectedNumFreight: 1,
		},
		{
			name:          "no freight collection in last promotion status should not panic",
			targetFreight: kargoapi.FreightReference{Name: "target-freight"},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}, {}},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							FreightCollection: nil,
						},
					},
				},
			},
			expectedNumFreight: 1,
		},
		{
			name:          "nil freight map in last promo collection should not panic",
			targetFreight: kargoapi.FreightReference{Name: "target-freight"},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}, {}},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							FreightCollection: &kargoapi.FreightCollection{
								Freight: nil,
							},
						},
					},
				},
			},
			expectedNumFreight: 1,
		},
		{
			name:          "requested freight greater than 1 and last promotion also has freight",
			targetFreight: kargoapi.FreightReference{Name: "target-freight"},
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "name-1",
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "name-2",
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "last-promo",
						Status: &kargoapi.PromotionStatus{
							FreightCollection: &kargoapi.FreightCollection{
								Freight: map[string]kargoapi.FreightReference{
									"Warehouse/name-1": {Origin: kargoapi.FreightOrigin{
										Kind: kargoapi.FreightOriginKindWarehouse,
										Name: "name-1",
									}},
									"Warehouse/name-2": {Origin: kargoapi.FreightOrigin{
										Kind: kargoapi.FreightOriginKindWarehouse,
										Name: "name-2",
									}},
								},
							},
						},
					},
				},
			},
			expectedNumFreight: 3,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := new(reconciler)
			result := r.buildTargetFreightCollection(
				t.Context(),
				tc.targetFreight,
				tc.stage,
			)
			require.Len(t, result.Freight, tc.expectedNumFreight)
		})
	}
}
