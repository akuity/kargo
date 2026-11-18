package stages

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"github.com/akuity/kargo/pkg/api/freightapi"
	"github.com/akuity/kargo/pkg/client/apiserver"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient/testutil"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	fakeevent "github.com/akuity/kargo/pkg/event/fake"
)

// markCall records the arguments of a single MarkFreightInStage call, so tests
// can assert on how the code under test drove the Freight API server client.
type markCall struct {
	project    string
	freight    string
	stage      string
	verified   bool
	verifiedAt time.Time
}

func TestControlFlowStageReconciler_Reconcile(t *testing.T) {
	testProject := "test-project"
	const testWarehouseName = "test-warehouse"
	testStageName := "test-stage"
	testStage := types.NamespacedName{
		Namespace: testProject,
		Name:      testStageName,
	}
	testStageMeta := metav1.ObjectMeta{
		Namespace: testProject,
		Name:      testStageName,
		Labels: map[string]string{
			kargoapi.LabelKeyShard: "test-shard",
		},
		Finalizers: []string{kargoapi.FinalizerName},
	}
	testWarehouseOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: testWarehouseName,
	}

	tests := []struct {
		name                    string
		req                     ctrl.Request
		stage                   *kargoapi.Stage
		interceptor             interceptor.Funcs
		listErr                 error
		listReferencingStageErr error
		assertions              func(*testing.T, client.Client, ctrl.Result, error)
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
			name: "shard mismatch",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testProject,
					Name:      testStageName,
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					PromotionTemplate: nil,
				},
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
					Labels: map[string]string{
						kargoapi.LabelKeyShard: "test-shard",
					},
				},
			},
			// The deletion cleanup lists Freight referencing the Stage through the
			// API server, so failing that surfaces the error.
			listReferencingStageErr: errors.New("something went wrong"),
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
					Labels: map[string]string{
						kargoapi.LabelKeyShard: "test-shard",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)

				// Verify finalizer was added
				stage := &kargoapi.Stage{}
				err = c.Get(t.Context(), testStage, stage)
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
					Labels: map[string]string{
						kargoapi.LabelKeyShard: "test-shard",
					},
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
				err = c.Get(t.Context(), testStage, stage)
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
			// Forces an error when attempting to list available Freight.
			listErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, ctrl.Result{}, result)

				// Verify error is recorded in status
				stage := &kargoapi.Stage{}
				err = c.Get(t.Context(), testStage, stage)
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
			// Forces an error when attempting to list available Freight.
			listErr: errors.New("something went wrong"),
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
				err = c.Get(t.Context(), testStage, stage)
				require.NoError(t, err)
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []client.Object
			if tt.stage != nil {
				objects = append(objects, tt.stage)
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(&kargoapi.Stage{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			// Freight reads and writes go through the API server client, not c.
			// Available Freight is resolved through the same client; tt.listErr
			// simulates the server failing that lookup.
			mockClient := &testutil.MockFreightClient{
				ListFreightByCurrentStageFun: func(context.Context, string, string) ([]kargoapi.Freight, error) {
					return nil, nil
				},
				ListFreightReferencingStageFun: func(context.Context, string, string) ([]kargoapi.Freight, error) {
					return nil, tt.listReferencingStageErr
				},
				ListFreightFromWarehouseFun: func(
					context.Context,
					string, string,
					*freightapi.ListWarehouseFreightOptions,
				) ([]kargoapi.Freight, error) {
					return nil, tt.listErr
				},
				MarkFreightInStageFun: func(
					context.Context, string, string, string, bool, time.Time,
				) error {
					return nil
				},
				ClearFreightStageStatusFun: func(context.Context, string, string, string) error {
					return nil
				},
			}

			r := &ControlFlowStageReconciler{
				client:        c,
				eventSender:   &fakeevent.Sender{},
				freightClient: mockClient,
				shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
					IsDefaultController: false,
					ShardName:           "test-shard",
				},
			}

			result, err := r.Reconcile(t.Context(), tt.req)
			tt.assertions(t, c, result, err)
		})
	}
}

func TestControlFlowStageReconciler_reconcile(t *testing.T) {
	const testProject = "test-project"
	const testWarehouseName = "test-warehouse"
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
		name      string
		stage     *kargoapi.Stage
		available []kargoapi.Freight
		// currentlyIn is what the API server reports as currently in the Stage. Any
		// of these no longer available to the Stage is unmarked (its CurrentlyIn
		// cleared).
		currentlyIn []kargoapi.Freight
		listErr     error
		markErr     error
		assertions  func(*testing.T, kargoapi.StageStatus, []markCall, []unmarkCall, error)
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
			// No Freight available from this Warehouse.
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
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
			available: []kargoapi.Freight{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "fake-freight",
				},
				Origin: testWarehouseOrigin,
			}},
			assertions: func(t *testing.T, status kargoapi.StageStatus, markCalls []markCall, _ []unmarkCall, err error) {
				require.NoError(t, err)
				// The Stage marks the available Freight as verified through the API
				// server.
				require.Equal(t, []markCall{
					{testProject, "fake-freight", testStage, true, markCalls[0].verifiedAt},
				}, markCalls)

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
			listErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
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
			available: []kargoapi.Freight{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProject,
					Name:      "fake-freight",
				},
				Origin: testWarehouseOrigin,
			}},
			markErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
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
			available: []kargoapi.Freight{{
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
			}},
			// A guard: already-verified Freight must be skipped, so MarkFreightInStage
			// must not be called. If it were, this error would surface.
			markErr: errors.New("should not be called"),
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []unmarkCall, err error) {
				require.NoError(t, err)
				assert.Len(t, status.Conditions, 1)
				assert.Equal(t, kargoapi.ConditionTypeReady, status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, status.Conditions[0].Status)
				assert.Equal(t, kargoapi.ConditionTypeReady, status.Conditions[0].Reason)
			},
		},
		{
			name: "unmarks Freight no longer available",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Sources: kargoapi.FreightSources{Direct: true},
						Origin:  testWarehouseOrigin,
					}},
				},
			},
			// Only new-freight is still available to the Stage.
			available: []kargoapi.Freight{{
				ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "new-freight"},
				Origin:     testWarehouseOrigin,
			}},
			// The API server reports new-freight (still available) and two old
			// Freight (no longer available) as currently in the Stage.
			currentlyIn: []kargoapi.Freight{
				{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "new-freight"}},
				{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "old-freight-1"}},
				{ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: "old-freight-2"}},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, unmarkCalls []unmarkCall, err error) {
				require.NoError(t, err)

				// Only the Freight no longer available is unmarked; new-freight is
				// still available and left alone.
				require.Equal(t, []unmarkCall{
					{testProject, "old-freight-1", testStage},
					{testProject, "old-freight-2", testStage},
				}, unmarkCalls)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var markCalls []markCall
			var unmarkCalls []unmarkCall
			mockClient := &testutil.MockFreightClient{
				ListFreightByCurrentStageFun: func(context.Context, string, string) ([]kargoapi.Freight, error) {
					return tt.currentlyIn, nil
				},
				ListFreightFromWarehouseFun: func(
					context.Context,
					string, string,
					*freightapi.ListWarehouseFreightOptions,
				) ([]kargoapi.Freight, error) {
					return tt.available, tt.listErr
				},
				MarkFreightInStageFun: func(
					_ context.Context,
					project, freight, stage string,
					verified bool,
					verifiedAt time.Time,
				) error {
					markCalls = append(markCalls, markCall{project, freight, stage, verified, verifiedAt})
					return tt.markErr
				},
				UnmarkFreightInStageFun: func(_ context.Context, project, freight, stage string) error {
					unmarkCalls = append(unmarkCalls, unmarkCall{project, freight, stage})
					return nil
				},
			}

			r := &ControlFlowStageReconciler{
				eventSender:   &fakeevent.Sender{},
				freightClient: mockClient,
				shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
					IsDefaultController: false,
					ShardName:           "test-shard",
				},
			}

			status, err := r.reconcile(t.Context(), tt.stage, time.Now())
			tt.assertions(t, status, markCalls, unmarkCalls, err)
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
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/fake-warehouse": {FreightName: "fake-freight"},
					},
				},
			},
			assertions: func(t *testing.T, newStatus kargoapi.StageStatus) {
				assert.Empty(t, newStatus.FreightHistory)
				assert.Nil(t, newStatus.Health)
				assert.Nil(t, newStatus.CurrentPromotion)
				assert.Nil(t, newStatus.LastPromotion)
				assert.Equal(t, "N/A", newStatus.FreightSummary)
				assert.Nil(t, newStatus.AutoPromotionHolds)
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
	justNow := time.Now()
	oneHourAgo := justNow.Add(-time.Hour)
	oneMinuteAgo := justNow.Add(-time.Minute)

	tests := []struct {
		name          string
		stage         *kargoapi.Stage
		freight       []kargoapi.Freight
		startTime     time.Time
		finishTime    time.Time
		markErrByName map[string]error
		assertions    func(*testing.T, []markCall, *fakeevent.Sender, error)
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
			assertions: func(t *testing.T, _ []markCall, recorder *fakeevent.Sender, err error) {
				require.NoError(t, err)
				assert.Len(t, recorder.Sent(), 0)
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
			assertions: func(t *testing.T, _ []markCall, recorder *fakeevent.Sender, err error) {
				require.NoError(t, err)
				assert.Len(t, recorder.Sent(), 0)
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
			assertions: func(t *testing.T, markCalls []markCall, recorder *fakeevent.Sender, err error) {
				require.NoError(t, err)
				assert.Len(t, recorder.Sent(), 2)

				// Both Freight are marked verified in the Stage at the finish time.
				require.Equal(t, []markCall{
					{"default", "freight-1", "test-stage", true, justNow},
					{"default", "freight-2", "test-stage", true, justNow},
				}, markCalls)
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
			assertions: func(t *testing.T, _ []markCall, recorder *fakeevent.Sender, err error) {
				require.NoError(t, err)
				require.Len(t, recorder.Sent(), 1)

				sent := recorder.Sent()[0]
				assert.Equal(
					t,
					string(kargoapi.EventTypeFreightVerificationSucceeded),
					sent.Type(),
				)
				assert.Equal(t, "Freight", kargoEvent.KindOf(sent))
				assert.Equal(
					t,
					"akuity.kargo.events.freight.verification.succeeded",
					sent.Subject(),
				)

				var data kargoEvent.FreightVerificationSucceeded
				require.NoError(t, sent.DataAs(&data))
				assert.Equal(t, "Freight verification succeeded", data.Message)
				require.NotNil(t, data.Actor)
				assert.Equal(t, "controller:stage-controller", *data.Actor)
				assert.Equal(t, "default", data.Project)
				assert.Equal(t, "test-stage", data.StageName)
				require.NotNil(t, data.Alias)
				assert.Equal(t, "fake-alias", *data.Alias)
				assert.Equal(t, "freight-1", data.Name)
				assert.Equal(
					t,
					oneHourAgo.Format(time.RFC3339),
					data.CreateTime.Format(time.RFC3339),
				)
				require.NotNil(t, data.StartTime)
				assert.Equal(
					t,
					oneMinuteAgo.Format(time.RFC3339),
					data.StartTime.Format(time.RFC3339),
				)
				require.NotNil(t, data.FinishTime)
				assert.Equal(
					t,
					justNow.Format(time.RFC3339),
					data.FinishTime.Format(time.RFC3339),
				)
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
			// freight-2 fails with a generic error (counted), freight-4 fails with a
			// NotFound (tolerated, not counted, because the Freight is gone).
			markErrByName: map[string]error{
				"freight-2": errors.New("something went wrong"),
				"freight-4": &apiserver.APIError{StatusCode: http.StatusNotFound},
			},
			assertions: func(t *testing.T, markCalls []markCall, recorder *fakeevent.Sender, err error) {
				require.ErrorContains(t, err, "failed to verify 1 Freight")

				assert.Len(t, recorder.Sent(), 2)

				// Every Freight is attempted, in order. freight-2 and freight-4 fail
				// their mark (so send no event); freight-1 and freight-3 succeed.
				names := make([]string, len(markCalls))
				for i, c := range markCalls {
					names[i] = c.freight
				}
				assert.Equal(t, []string{"freight-1", "freight-2", "freight-3", "freight-4"}, names)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var markCalls []markCall
			mockClient := &testutil.MockFreightClient{
				MarkFreightInStageFun: func(
					_ context.Context,
					project, freight, stage string,
					verified bool,
					verifiedAt time.Time,
				) error {
					markCalls = append(markCalls, markCall{project, freight, stage, verified, verifiedAt})
					return tt.markErrByName[freight]
				},
			}
			recorder := &fakeevent.Sender{}

			r := &ControlFlowStageReconciler{
				eventSender:   recorder,
				freightClient: mockClient,
			}

			_, err := r.markFreightVerifiedForStage(t.Context(), tt.stage, tt.freight, tt.startTime, tt.finishTime)
			tt.assertions(t, markCalls, recorder, err)
		})
	}
}

func TestControlFlowStageReconciler_handleDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name                    string
		stage                   *kargoapi.Stage
		freight                 []*kargoapi.Freight
		interceptor             interceptor.Funcs
		listReferencingStageErr error
		assertions              func(*testing.T, []clearCall, *kargoapi.Stage, error)
	}{
		{
			name: "finalizer already removed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
			},
			// A guard: with no finalizer, handleDelete returns before clearing any
			// Freight status, so this error must not surface.
			listReferencingStageErr: errors.New("unexpected call to clear Freight status"),
			assertions: func(t *testing.T, _ []clearCall, _ *kargoapi.Stage, err error) {
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
			assertions: func(t *testing.T, _ []clearCall, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				assert.False(t, controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName))
			},
		},
		{
			name: "clears Freight status and removes finalizer",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace:  "default",
					Finalizers: []string{kargoapi.FinalizerName},
				},
			},
			freight: []*kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "verified"},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{"test-stage": {}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "approved"},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{"test-stage": {}},
					},
				},
			},
			assertions: func(t *testing.T, clearCalls []clearCall, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				assert.False(t, controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName))

				// Each referencing Freight is cleared of the Stage.
				assert.Equal(t, []clearCall{
					{"default", "verified", "test-stage"},
					{"default", "approved", "test-stage"},
				}, clearCalls)
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
			listReferencingStageErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, _ []clearCall, stage *kargoapi.Stage, err error) {
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
			assertions: func(t *testing.T, _ []clearCall, _ *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "failed to remove finalizer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.stage).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			referencing := make([]kargoapi.Freight, len(tt.freight))
			for i, f := range tt.freight {
				referencing[i] = *f
			}
			var clearCalls []clearCall
			mockClient := &testutil.MockFreightClient{
				ListFreightReferencingStageFun: func(context.Context, string, string) ([]kargoapi.Freight, error) {
					return referencing, tt.listReferencingStageErr
				},
				ClearFreightStageStatusFun: func(_ context.Context, project, freight, stage string) error {
					clearCalls = append(clearCalls, clearCall{project, freight, stage})
					return nil
				},
			}

			r := &ControlFlowStageReconciler{
				client:        c,
				freightClient: mockClient,
			}

			err := r.handleDelete(t.Context(), tt.stage)
			tt.assertions(t, clearCalls, tt.stage, err)
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
				err = c.List(t.Context(), &runs,
					client.InNamespace("default"),
					client.MatchingLabels{kargoapi.LabelKeyStage: "test-stage"},
				)
				require.NoError(t, err)
				assert.Empty(t, runs.Items)

				// Verify other analysis runs still exist
				err = c.List(t.Context(), &runs,
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
				err = c.List(t.Context(), &runs, client.InNamespace("default"))
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

			tt.assertions(t, c, clearAnalysisRuns(t.Context(), r.newVerifier(), tt.stage))
		})
	}
}
