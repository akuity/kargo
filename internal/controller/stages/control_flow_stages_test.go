package stages

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/indexer"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
)

func TestControlFlowStageReconciler_Reconcile(t *testing.T) {
	testProject := "test-project"
	const testWarehouseName = "test-warehouse"
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      testWarehouseName,
		},
	}
	testStageName := "test-stage"
	testStage := types.NamespacedName{
		Namespace: testProject,
		Name:      testStageName,
	}
	testStageMeta := metav1.ObjectMeta{
		Namespace:  testProject,
		Name:       testStageName,
		Finalizers: []string{kargoapi.FinalizerName},
	}
	testWarehouseOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: testWarehouseName,
	}

	tests := []struct {
		name        string
		req         ctrl.Request
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, ctrl.Result, error)
	}{
		{
			name: "stage not found",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testProject,
					Name:      "non-existent",
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "ignores non-control flow stage",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					// Not a control flow stage
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{}},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "handles deletion",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testProject,
					Name:              testStageName,
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{kargoapi.FinalizerName},
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "deletion error",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         testProject,
					Name:              testStageName,
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{kargoapi.FinalizerName},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "adds finalizer and requeues",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      testStageName,
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)

				// Verify finalizer was added
				stage := &kargoapi.Stage{}
				err = c.Get(context.Background(), testStage, stage)
				require.NoError(t, err)
				assert.Contains(t, stage.Finalizers, kargoapi.FinalizerName)
			},
		},
		{
			name: "removes stale annotations",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  testProject,
					Name:       testStageName,
					Finalizers: []string{kargoapi.FinalizerName},
					Annotations: map[string]string{
						kargoapi.AnnotationKeyArgoCDContext: "old-argocd-context",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)

				// Verify annotation was removed
				stage := &kargoapi.Stage{}
				err = c.Get(context.Background(), testStage, stage)
				require.NoError(t, err)
				assert.NotContains(t, stage.Annotations, kargoapi.AnnotationKeyArgoCDContext)
			},
		},
		{
			name: "reconcile error",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin:  testWarehouseOrigin,
						Sources: kargoapi.FreightSources{Direct: true},
					}},
				},
			},
			objects: []client.Object{testWarehouse},
			interceptor: interceptor.Funcs{
				// This will force an error when attempting to list available Freight
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, ctrl.Result{}, result)

				// Verify error is recorded in status
				stage := &kargoapi.Stage{}
				err = c.Get(context.Background(), testStage, stage)
				require.NoError(t, err)
			},
		},
		{
			name: "status update error after reconcile error",
			req:  ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin:  testWarehouseOrigin,
						Sources: kargoapi.FreightSources{Direct: true},
					}},
				},
			},
			objects: []client.Object{testWarehouse},
			interceptor: interceptor.Funcs{
				// This will force an error when attempting to list available Freight
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
				// This will force an error when attempting to update the Stage status
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return fmt.Errorf("status update error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				// Should return the reconcile error, not the status update error
				require.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name:  "status update error after successful reconcile",
			req:   ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{ObjectMeta: testStageMeta},
			interceptor: interceptor.Funcs{
				// This will force an error when attempting to update the Stage status
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return fmt.Errorf("status update error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.ErrorContains(t, err, "failed to update Stage status: status update error")
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name:  "successful reconciliation",
			req:   ctrl.Request{NamespacedName: testStage},
			stage: &kargoapi.Stage{ObjectMeta: testStageMeta},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)

				// Verify status was updated
				stage := &kargoapi.Stage{}
				err = c.Get(context.Background(), testStage, stage)
				require.NoError(t, err)
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := tt.objects
			if tt.stage != nil {
				objects = append(objects, tt.stage)
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByWarehouseField,
					indexer.FreightByWarehouse,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesField,
					indexer.FreightByVerifiedStages,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesField,
					indexer.FreightApprovedForStages,
				).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &ControlFlowStageReconciler{
				client:        c,
				eventRecorder: fakeevent.NewEventRecorder(10),
			}

			result, err := r.Reconcile(context.Background(), tt.req)
			tt.assertions(t, c, result, err)
		})
	}
}

func TestControlFlowStageReconciler_reconcile(t *testing.T) {
	const testProject = "test-project"
	const testWarehouseName = "test-warehouse"
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      testWarehouseName,
		},
	}
	const testStage = "test-stage"
	testStageMeta := metav1.ObjectMeta{
		Namespace: testProject,
		Name:      testStage,
	}
	testWarehouseOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: testWarehouseName,
	}

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, kargoapi.StageStatus, client.Client, error)
	}{
		{
			name: "no available Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Sources: kargoapi.FreightSources{Direct: true},
						Origin:  testWarehouseOrigin,
					}},
				},
			},
			objects: []client.Object{
				testWarehouse,
				// No Freight exists from this Warehouse
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.NoError(t, err)

				require.Len(t, status.Conditions, 1)
				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
			},
		},
		{
			name: "available Freight not yet marked as verified",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Sources: kargoapi.FreightSources{Direct: true},
						Origin:  testWarehouseOrigin,
					}},
				},
			},
			objects: []client.Object{
				testWarehouse,
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: testWarehouseOrigin,
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, c client.Client, err error) {
				require.NoError(t, err)
				updatedFreight := &kargoapi.Freight{}
				err = c.Get(
					context.Background(),
					types.NamespacedName{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					updatedFreight,
				)
				require.NoError(t, err)
				assert.Contains(t, updatedFreight.Status.VerifiedIn, testStage)

				require.Len(t, status.Conditions, 1)
				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
			},
		},
		{
			name: "error listing available Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Sources: kargoapi.FreightSources{Direct: true},
						Origin:  testWarehouseOrigin,
					}},
				},
			},
			interceptor: interceptor.Funcs{
				// Listing Freight begins with a Get call to the Warehouse, so this
				// will force an error
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")

				require.Len(t, status.Conditions, 2)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "FreightRetrievalFailed", readyCond.Reason)
				assert.Contains(t, readyCond.Message, "something went wrong")

				recCond := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, recCond)
				assert.Equal(t, metav1.ConditionTrue, recCond.Status)
				assert.Equal(t, "RetryAfterFreightRetrievalFailed", recCond.Reason)
				assert.Contains(t, recCond.Message, "something went wrong")
			},
		},
		{
			name: "error marking Freight as verified",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Sources: kargoapi.FreightSources{Direct: true},
						Origin:  testWarehouseOrigin,
					}},
				},
			},
			objects: []client.Object{
				testWarehouse,
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: testWarehouseOrigin,
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.ErrorContains(t, err, "failed to verify 1 Freight")

				require.Len(t, status.Conditions, 2)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "FreightVerificationFailed", readyCond.Reason)
				assert.Contains(t, readyCond.Message, "failed to verify 1 Freight")

				recCond := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, recCond)
				assert.Equal(t, metav1.ConditionTrue, recCond.Status)
				assert.Equal(t, "RetryAfterVerificationFailed", recCond.Reason)
				assert.Contains(t, recCond.Message, "failed to verify 1 Freight")
			},
		},
		{
			name: "already verified freight",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Sources: kargoapi.FreightSources{Direct: true},
						Origin:  testWarehouseOrigin,
					}},
				},
			},
			objects: []client.Object{
				testWarehouse,
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight",
					},
					Origin: testWarehouseOrigin,
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage: {},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				// This is intended to force an error if there is an unexpected attempt
				// to patch the Freight status, which should not happen if the Freight
				// is already marked as verified.
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.NoError(t, err)

				require.Len(t, status.Conditions, 1)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
			},
		},
		{
			name: "handles refresh annotation",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testStage,
					Namespace: testProject,
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "refresh-token",
					},
				},
			},
			objects: []client.Object{testWarehouse},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.NoError(t, err)
				assert.Equal(t, "refresh-token", status.LastHandledRefresh)
			},
		},
		{
			name: "observes generation on reconciliation",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       testStage,
					Namespace:  testProject,
					Generation: 2,
				},
				Status: kargoapi.StageStatus{
					ObservedGeneration: 1,
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.NoError(t, err)
				assert.Equal(t, int64(2), status.ObservedGeneration)
			},
		},
		{
			name: "removes reconciling condition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testStage,
					Namespace: testProject,
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{
						{
							Type:   kargoapi.ConditionTypeReconciling,
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ client.Client, err error) {
				require.NoError(t, err)
				assert.Len(t, status.Conditions, 1)
				assert.Equal(t, kargoapi.ConditionTypeReady, status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, status.Conditions[0].Status)
				assert.Equal(t, kargoapi.ConditionTypeReady, status.Conditions[0].Reason)
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByWarehouseField,
					indexer.FreightByWarehouse,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesField,
					indexer.FreightByVerifiedStages,
				).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &ControlFlowStageReconciler{
				client:        c,
				eventRecorder: fakeevent.NewEventRecorder(10),
			}

			status, err := r.reconcile(context.Background(), tt.stage, time.Now())
			tt.assertions(t, status, c, err)
		})
	}
}

func TestControlFlowStageReconciler_initializeStatus(t *testing.T) {
	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		assertions func(*testing.T, kargoapi.StageStatus)
	}{
		{
			name: "initializes status",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
			},
			assertions: func(t *testing.T, newStatus kargoapi.StageStatus) {
				assert.Equal(t, int64(2), newStatus.ObservedGeneration)
			},
		},
		{
			name: "records refresh token",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "refresh-token",
					},
				},
			},
			assertions: func(t *testing.T, newStatus kargoapi.StageStatus) {
				assert.Equal(t, "refresh-token", newStatus.LastHandledRefresh)
			},
		},
		{
			name: "clears irrelevant conditions for Stage type",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{
						{
							// Should be kept
							Type:   kargoapi.ConditionTypeReady,
							Status: metav1.ConditionTrue,
						},
						{
							// Should be kept
							Type:   kargoapi.ConditionTypeReconciling,
							Status: metav1.ConditionTrue,
						},
						{
							// Should be cleared
							Type:   kargoapi.ConditionTypeHealthy,
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			assertions: func(t *testing.T, newStatus kargoapi.StageStatus) {
				assert.Len(t, newStatus.Conditions, 2)
				assert.Equal(t, kargoapi.ConditionTypeReady, newStatus.Conditions[0].Type)
				assert.Equal(t, kargoapi.ConditionTypeReconciling, newStatus.Conditions[1].Type)
			},
		},
		{
			name: "clears irrelevant fields for Stage type",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory:   kargoapi.FreightHistory{{}, {}, {}},
					Health:           &kargoapi.Health{},
					CurrentPromotion: &kargoapi.PromotionReference{},
					LastPromotion:    &kargoapi.PromotionReference{},
					FreightSummary:   "old freight summary",
				},
			},
			assertions: func(t *testing.T, newStatus kargoapi.StageStatus) {
				assert.Empty(t, newStatus.FreightHistory)
				assert.Nil(t, newStatus.Health)
				assert.Nil(t, newStatus.CurrentPromotion)
				assert.Nil(t, newStatus.LastPromotion)
				assert.Equal(t, "N/A", newStatus.FreightSummary)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newStatus := (&ControlFlowStageReconciler{}).initializeStatus(tt.stage)
			tt.assertions(t, newStatus)
		})
	}
}

func TestControlFlowStageReconciler_markFreightVerifiedForStage(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	toObjSlice := func(freights []kargoapi.Freight) []client.Object {
		ptrs := make([]client.Object, len(freights))
		for i, f := range freights {
			ptrs[i] = f.DeepCopy()
		}
		return ptrs
	}

	justNow := time.Now()
	oneHourAgo := justNow.Add(-time.Hour)
	oneMinuteAgo := justNow.Add(-time.Minute)

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		freight     []kargoapi.Freight
		startTime   time.Time
		finishTime  time.Time
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, *fakeevent.EventRecorder, error)
	}{
		{
			name: "no freight to verify",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			freight: nil,
			assertions: func(t *testing.T, _ client.Client, recorder *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				assert.Len(t, recorder.Events, 0)
			},
		},
		{
			name: "freight has already been verified",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			freight: []kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, recorder *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				assert.Len(t, recorder.Events, 0)
			},
		},
		{
			name: "verifies freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			freight: []kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-2",
					},
				},
			},
			finishTime: justNow,
			assertions: func(t *testing.T, c client.Client, recorder *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				assert.Len(t, recorder.Events, 2)

				freight1 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-1",
				}, freight1))
				assert.Contains(t, freight1.Status.VerifiedIn, "test-stage")
				assert.Equal(
					t,
					justNow.Unix(),
					freight1.Status.VerifiedIn["test-stage"].VerifiedAt.Unix(),
				)

				freight2 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-2",
				}, freight2))
				assert.Contains(t, freight2.Status.VerifiedIn, "test-stage")
				assert.Equal(
					t,
					justNow.Unix(),
					freight2.Status.VerifiedIn["test-stage"].VerifiedAt.Unix(),
				)
			},
		},
		{
			name: "records event for verified freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			freight: []kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
						CreationTimestamp: metav1.Time{
							Time: oneHourAgo,
						},
					},
					Alias: "fake-alias",
				},
			},
			startTime:  oneMinuteAgo,
			finishTime: justNow,
			assertions: func(t *testing.T, _ client.Client, recorder *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events

				assert.Equal(t, corev1.EventTypeNormal, event.EventType)
				assert.Equal(t, kargoapi.EventReasonFreightVerificationSucceeded, event.Reason)
				assert.Equal(t, "Freight verification succeeded", event.Message)

				assert.Equal(t, map[string]string{
					kargoapi.AnnotationKeyEventActor:                  "controller:stage-controller",
					kargoapi.AnnotationKeyEventProject:                "default",
					kargoapi.AnnotationKeyEventStageName:              "test-stage",
					kargoapi.AnnotationKeyEventFreightAlias:           "fake-alias",
					kargoapi.AnnotationKeyEventFreightName:            "freight-1",
					kargoapi.AnnotationKeyEventFreightCreateTime:      oneHourAgo.Format(time.RFC3339),
					kargoapi.AnnotationKeyEventVerificationStartTime:  oneMinuteAgo.Format(time.RFC3339),
					kargoapi.AnnotationKeyEventVerificationFinishTime: justNow.Format(time.RFC3339),
				}, event.Annotations)
			},
		},
		{
			name: "continues on patch error for freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			freight: []kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-3",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-4",
					},
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					ctx context.Context,
					client client.Client,
					subResourceName string,
					obj client.Object,
					patch client.Patch,
					opts ...client.SubResourcePatchOption,
				) error {
					switch obj.GetName() {
					case "freight-2":
						return fmt.Errorf("something went wrong")
					case "freight-4":
						return apierrors.NewNotFound(
							kargoapi.GroupVersion.WithResource("freight").GroupResource(), "freight-4",
						)
					default:
						return client.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
					}
				},
			},
			assertions: func(t *testing.T, c client.Client, recorder *fakeevent.EventRecorder, err error) {
				require.ErrorContains(t, err, "failed to verify 1 Freight")

				assert.Len(t, recorder.Events, 2)

				freight1 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-1",
				}, freight1))
				assert.Contains(t, freight1.Status.VerifiedIn, "test-stage")

				freight2 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-2",
				}, freight2))
				assert.NotContains(t, freight2.Status.VerifiedIn, "test-stage")

				freight3 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-3",
				}, freight3))
				assert.Contains(t, freight3.Status.VerifiedIn, "test-stage")

				freight4 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-4",
				}, freight4))
				assert.NotContains(t, freight4.Status.VerifiedIn, "test-stage")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(toObjSlice(tt.freight)...).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()
			recorder := fakeevent.NewEventRecorder(10)

			r := &ControlFlowStageReconciler{
				client:        c,
				eventRecorder: recorder,
			}

			_, err := r.markFreightVerifiedForStage(context.Background(), tt.stage, tt.freight, tt.startTime, tt.finishTime)
			tt.assertions(t, c, recorder, err)
		})
	}
}

func TestControlFlowStageReconciler_handleDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *kargoapi.Stage, error)
	}{
		{
			name: "finalizer already removed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("unexpected call to List")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "removes finalizer",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace:  "default",
					Finalizers: []string{kargoapi.FinalizerName},
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				assert.False(t, controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName))
			},
		},
		{
			name: "does not remove finalizer on cleanup error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace:  "default",
					Finalizers: []string{kargoapi.FinalizerName},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(
					ctx context.Context,
					c client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					listOpts := &client.ListOptions{}
					for _, opt := range opts {
						opt.ApplyToList(listOpts)
					}

					switch {
					case strings.Contains(listOpts.FieldSelector.String(), indexer.FreightApprovedForStagesField):
						return fmt.Errorf("something went wrong")
					default:
						return c.List(ctx, list, opts...)
					}
				},
			},
			assertions: func(t *testing.T, stage *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.True(t, controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName))
			},
		},
		{
			name: "finalizer removal error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace:  "default",
					Finalizers: []string{kargoapi.FinalizerName},
				},
			},
			interceptor: interceptor.Funcs{
				Patch: func(
					context.Context,
					client.WithWatch,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return fmt.Errorf("failed to remove finalizer")
				},
			},
			assertions: func(t *testing.T, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "failed to remove finalizer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.stage).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesField,
					indexer.FreightByVerifiedStages,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesField,
					indexer.FreightApprovedForStages,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &ControlFlowStageReconciler{
				client: c,
			}

			err := r.handleDelete(context.Background(), tt.stage)
			tt.assertions(t, tt.stage, err)
		})
	}
}

func TestControlFlowStageReconciler_clearVerifications(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, error)
	}{
		{
			name: "no freight to clear",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "clears verifications from multiple freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage":    {},
							"another-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-2",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				freight1 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-1",
				}, freight1))
				assert.NotContains(t, freight1.Status.VerifiedIn, "test-stage")
				assert.Contains(t, freight1.Status.VerifiedIn, "another-stage")

				freight2 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-2",
				}, freight2))
				assert.Empty(t, freight2.Status.VerifiedIn)
			},
		},
		{
			name: "handles listing error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("listing error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "listing error")
			},
		},
		{
			name: "continues on patch error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "ignores not found errors on patch",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return apierrors.NewNotFound(
						kargoapi.GroupVersion.WithResource("freight").GroupResource(),
						"freight-1",
					)
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesField,
					indexer.FreightByVerifiedStages,
				).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &ControlFlowStageReconciler{
				client: c,
			}

			tt.assertions(t, c, r.clearVerifications(context.Background(), tt.stage))
		})
	}
}

func TestControlFlowStageReconciler_clearApprovals(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, error)
	}{
		{
			name: "no freight to clear",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "clears approvals from multiple freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage":    {},
							"another-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-2",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				freight1 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-1",
				}, freight1))
				assert.NotContains(t, freight1.Status.ApprovedFor, "test-stage")
				assert.Contains(t, freight1.Status.ApprovedFor, "another-stage")

				freight2 := &kargoapi.Freight{}
				require.NoError(t, c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "freight-2",
				}, freight2))
				assert.Empty(t, freight2.Status.ApprovedFor)
			},
		},
		{
			name: "handles listing error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("listing error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "listing error")
			},
		},
		{
			name: "continues on patch error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "ignores not found errors on patch",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					context.Context,
					client.Client,
					string,
					client.Object,
					client.Patch,
					...client.SubResourcePatchOption,
				) error {
					return apierrors.NewNotFound(
						kargoapi.GroupVersion.WithResource("freight").GroupResource(),
						"freight-1",
					)
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesField,
					indexer.FreightApprovedForStages,
				).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &ControlFlowStageReconciler{
				client: c,
			}

			tt.assertions(t, c, r.clearApprovals(context.Background(), tt.stage))
		})
	}
}

func TestControlFlowStageReconciler_clearAnalysisRuns(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rollouts.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		cfg         ReconcilerConfig
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, error)
	}{
		{
			name: "rollouts integration disabled",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			cfg: ReconcilerConfig{
				RolloutsIntegrationEnabled: false,
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "deletes analysis runs",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			cfg: ReconcilerConfig{
				RolloutsIntegrationEnabled: true,
			},
			objects: []client.Object{
				&rollouts.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "analysis-1",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
				},
				&rollouts.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "analysis-2",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
				},
				&rollouts.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "analysis-other",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "other-stage",
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				// Verify analysis runs for test-stage are deleted
				var runs rollouts.AnalysisRunList
				err = c.List(context.Background(), &runs,
					client.InNamespace("default"),
					client.MatchingLabels{kargoapi.LabelKeyStage: "test-stage"},
				)
				require.NoError(t, err)
				assert.Empty(t, runs.Items)

				// Verify other analysis runs still exist
				err = c.List(context.Background(), &runs,
					client.InNamespace("default"),
					client.MatchingLabels{kargoapi.LabelKeyStage: "other-stage"},
				)
				require.NoError(t, err)
				assert.Len(t, runs.Items, 1)
				assert.Equal(t, "analysis-other", runs.Items[0].Name)
			},
		},
		{
			name: "handles deletion error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			cfg: ReconcilerConfig{
				RolloutsIntegrationEnabled: true,
			},
			objects: []client.Object{
				&rollouts.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "analysis-1",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				DeleteAllOf: func(context.Context, client.WithWatch, client.Object, ...client.DeleteAllOfOption) error {
					return fmt.Errorf("deletion error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "deletion error")
			},
		},
		{
			name: "no analysis runs to delete",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			cfg: ReconcilerConfig{
				RolloutsIntegrationEnabled: true,
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				var runs rollouts.AnalysisRunList
				err = c.List(context.Background(), &runs, client.InNamespace("default"))
				require.NoError(t, err)
				assert.Empty(t, runs.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &ControlFlowStageReconciler{
				client: c,
				cfg:    tt.cfg,
			}

			tt.assertions(t, c, r.clearAnalysisRuns(context.Background(), tt.stage))
		})
	}
}
