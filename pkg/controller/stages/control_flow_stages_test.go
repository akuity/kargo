package stages

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient/testutil"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	fakeevent "github.com/akuity/kargo/pkg/event/fake"
)

// markCall records the arguments of a single MarkFreightsCurrentlyInStage
// call, so tests can assert on how the code under test drove the Freight API
// server client. Freight names are recorded sorted, since the order in which
// they are sent carries no meaning.
type markCall struct {
	project string
	stage   string
	freight []string
}

// markVerifiedCall records the arguments of a single
// MarkFreightsVerifiedInStage call.
type markVerifiedCall struct {
	project    string
	stage      string
	freight    []string
	verifiedAt time.Time
}

// clearStageCall records the arguments of a single ClearStageFreights call.
type clearStageCall struct {
	project string
	stage   string
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
		name        string
		req         ctrl.Request
		stage       *kargoapi.Stage
		interceptor interceptor.Funcs
		listErr     error
		clearErr    error
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
			// The deletion cleanup clears the Stage from Freight status through the
			// API server, so failing that surfaces the error.
			clearErr: errors.New("something went wrong"),
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
				ListFreightFromOriginFun: func(
					context.Context,
					string,
					kargoapi.FreightOrigin,
					*freightclient.ListFreightFromOriginOptions,
				) ([]kargoapi.Freight, error) {
					return nil, tt.listErr
				},
				MarkFreightsCurrentlyInStageFun: func(context.Context, string, string, []string) error {
					return nil
				},
				MarkFreightsVerifiedInStageFun: func(
					context.Context,
					string, string,
					[]string,
					time.Time,
				) error {
					return nil
				},
				ClearStageFreightsFun: func(context.Context, string, string) error {
					return tt.clearErr
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

	otherWarehouseOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "other-warehouse",
	}
	directFrom := func(origins ...kargoapi.FreightOrigin) []kargoapi.FreightRequest {
		reqs := make([]kargoapi.FreightRequest, len(origins))
		for i, o := range origins {
			reqs[i] = kargoapi.FreightRequest{
				Sources: kargoapi.FreightSources{Direct: true},
				Origin:  o,
			}
		}
		return reqs
	}
	newFreight := func(
		name string,
		origin kargoapi.FreightOrigin,
		discoveredAt time.Time,
		verified bool,
	) kargoapi.Freight {
		f := kargoapi.Freight{
			ObjectMeta:   metav1.ObjectMeta{Namespace: testProject, Name: name},
			Origin:       origin,
			DiscoveredAt: &metav1.Time{Time: discoveredAt},
		}
		if verified {
			f.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{testStage: {}}
		}
		return f
	}
	now := time.Now()

	tests := []struct {
		name            string
		stage           *kargoapi.Stage
		available       []kargoapi.Freight
		listErr         error
		markErr         error
		markVerifiedErr error
		assertions      func(*testing.T, kargoapi.StageStatus, []markCall, []markVerifiedCall, error)
	}{
		{
			name: "no available Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec:       kargoapi.StageSpec{RequestedFreight: directFrom(testWarehouseOrigin)},
			},
			// No Freight available from this Warehouse.
			assertions: func(
				t *testing.T,
				status kargoapi.StageStatus,
				markCalls []markCall,
				markVerifiedCalls []markVerifiedCall,
				err error,
			) {
				require.NoError(t, err)

				// With nothing available, Freight status is left untouched.
				require.Empty(t, markCalls)
				require.Empty(t, markVerifiedCalls)

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
				Spec:       kargoapi.StageSpec{RequestedFreight: directFrom(testWarehouseOrigin)},
			},
			available: []kargoapi.Freight{
				newFreight("fake-freight", testWarehouseOrigin, now, false),
			},
			assertions: func(
				t *testing.T,
				status kargoapi.StageStatus,
				markCalls []markCall,
				markVerifiedCalls []markVerifiedCall,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, markVerifiedCalls, 1)
				require.Equal(t, testProject, markVerifiedCalls[0].project)
				require.Equal(t, testStage, markVerifiedCalls[0].stage)
				require.Equal(t, []string{"fake-freight"}, markVerifiedCalls[0].freight)

				require.Equal(t, []markCall{{testProject, testStage, []string{"fake-freight"}}}, markCalls)

				require.Len(t, status.Conditions, 1)
				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
			},
		},
		{
			name: "marks all unverified Freight verified and the newest per origin current",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec: kargoapi.StageSpec{
					RequestedFreight: directFrom(testWarehouseOrigin, otherWarehouseOrigin),
				},
			},
			available: []kargoapi.Freight{
				newFreight("wh-old", testWarehouseOrigin, now.Add(-time.Hour), false),
				newFreight("wh-new", testWarehouseOrigin, now, false),
				newFreight("other-verified", otherWarehouseOrigin, now.Add(-time.Hour), true),
				newFreight("other-new", otherWarehouseOrigin, now, false),
			},
			assertions: func(
				t *testing.T,
				status kargoapi.StageStatus,
				markCalls []markCall,
				markVerifiedCalls []markVerifiedCall,
				err error,
			) {
				require.NoError(t, err)

				// A single request verifies all available Freight not yet verified.
				require.Len(t, markVerifiedCalls, 1)
				require.Equal(t, []string{"other-new", "wh-new", "wh-old"}, markVerifiedCalls[0].freight)

				// A single request marks the newest Freight from each origin as
				// current; the API server unmarks any other Freight.
				require.Equal(t,
					[]markCall{{testProject, testStage, []string{"other-new", "wh-new"}}},
					markCalls,
				)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
			},
		},
		{
			name: "error listing available Freight",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec:       kargoapi.StageSpec{RequestedFreight: directFrom(testWarehouseOrigin)},
			},
			listErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []markVerifiedCall, err error) {
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
				Spec:       kargoapi.StageSpec{RequestedFreight: directFrom(testWarehouseOrigin)},
			},
			available: []kargoapi.Freight{
				newFreight("fake-freight", testWarehouseOrigin, now, false),
			},
			markVerifiedErr: errors.New("something went wrong"),
			assertions: func(
				t *testing.T,
				status kargoapi.StageStatus,
				markCalls []markCall,
				_ []markVerifiedCall,
				err error,
			) {
				require.ErrorContains(t, err, "failed to verify Freights")

				// Marking current is not attempted after verification fails.
				require.Empty(t, markCalls)

				require.Len(t, status.Conditions, 2)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "FreightVerificationFailed", readyCond.Reason)
				assert.Contains(t, readyCond.Message, "failed to verify Freights")

				recCond := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, recCond)
				assert.Equal(t, metav1.ConditionTrue, recCond.Status)
				assert.Equal(t, "RetryAfterVerificationFailed", recCond.Reason)
				assert.Contains(t, recCond.Message, "failed to verify Freights")
			},
		},
		{
			name: "error marking Freight as current",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec:       kargoapi.StageSpec{RequestedFreight: directFrom(testWarehouseOrigin)},
			},
			available: []kargoapi.Freight{
				newFreight("fake-freight", testWarehouseOrigin, now, false),
			},
			markErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []markVerifiedCall, err error) {
				require.ErrorContains(t, err, "something went wrong")

				require.Len(t, status.Conditions, 2)

				readyCond := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "FreightMarkCurrentFailed", readyCond.Reason)
				assert.Contains(t, readyCond.Message, "something went wrong")

				recCond := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, recCond)
				assert.Equal(t, metav1.ConditionTrue, recCond.Status)
				assert.Equal(t, "RetryAfterMarkCurrentFailed", recCond.Reason)
				assert.Contains(t, recCond.Message, "something went wrong")
			},
		},
		{
			name: "already verified freight",
			stage: &kargoapi.Stage{
				ObjectMeta: testStageMeta,
				Spec:       kargoapi.StageSpec{RequestedFreight: directFrom(testWarehouseOrigin)},
			},
			available: []kargoapi.Freight{
				newFreight("fake-freight", testWarehouseOrigin, now, true),
			},
			assertions: func(
				t *testing.T,
				status kargoapi.StageStatus,
				markCalls []markCall,
				markVerifiedCalls []markVerifiedCall,
				err error,
			) {
				require.NoError(t, err)

				// Already-verified Freight is not sent to be verified again.
				for _, c := range markVerifiedCalls {
					require.Empty(t, c.freight)
				}
				// It is still marked as current.
				require.Equal(t, []markCall{{testProject, testStage, []string{"fake-freight"}}}, markCalls)

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
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []markVerifiedCall, err error) {
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []markVerifiedCall, err error) {
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, _ []markCall, _ []markVerifiedCall, err error) {
				require.NoError(t, err)
				assert.Len(t, status.Conditions, 1)
				assert.Equal(t, kargoapi.ConditionTypeReady, status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, status.Conditions[0].Status)
				assert.Equal(t, kargoapi.ConditionTypeReady, status.Conditions[0].Reason)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var markCalls []markCall
			var markVerifiedCalls []markVerifiedCall
			mockClient := &testutil.MockFreightClient{
				ListFreightFromOriginFun: func(
					_ context.Context,
					_ string,
					origin kargoapi.FreightOrigin,
					_ *freightclient.ListFreightFromOriginOptions,
				) ([]kargoapi.Freight, error) {
					var out []kargoapi.Freight
					for _, f := range tt.available {
						if f.Origin == origin {
							out = append(out, f)
						}
					}
					return out, tt.listErr
				},
				MarkFreightsCurrentlyInStageFun: func(
					_ context.Context,
					project, stage string,
					freight []string,
				) error {
					markCalls = append(markCalls, markCall{project, stage, slices.Sorted(slices.Values(freight))})
					return tt.markErr
				},
				MarkFreightsVerifiedInStageFun: func(
					_ context.Context,
					project, stage string,
					freight []string,
					verifiedAt time.Time,
				) error {
					markVerifiedCalls = append(
						markVerifiedCalls,
						markVerifiedCall{project, stage, slices.Sorted(slices.Values(freight)), verifiedAt},
					)
					return tt.markVerifiedErr
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
			tt.assertions(t, status, markCalls, markVerifiedCalls, err)
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

	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-stage",
		},
	}
	unverified := func(name string) kargoapi.Freight {
		return kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name},
		}
	}
	verified := func(name string) kargoapi.Freight {
		f := unverified(name)
		f.Status.VerifiedIn = map[string]kargoapi.VerifiedStage{"test-stage": {}}
		return f
	}

	tests := []struct {
		name       string
		freight    []kargoapi.Freight
		startTime  time.Time
		finishTime time.Time
		markErr    error
		assertions func(*testing.T, []markVerifiedCall, *fakeevent.Sender, int, error)
	}{
		{
			name:    "no freight to verify",
			freight: nil,
			assertions: func(
				t *testing.T,
				markCalls []markVerifiedCall,
				recorder *fakeevent.Sender,
				count int,
				err error,
			) {
				require.NoError(t, err)
				require.Zero(t, count)
				for _, c := range markCalls {
					require.Empty(t, c.freight)
				}
				assert.Empty(t, recorder.Sent())
			},
		},
		{
			name:    "freight has already been verified",
			freight: []kargoapi.Freight{verified("freight-1")},
			assertions: func(
				t *testing.T,
				markCalls []markVerifiedCall,
				recorder *fakeevent.Sender,
				count int,
				err error,
			) {
				require.NoError(t, err)
				require.Zero(t, count)
				for _, c := range markCalls {
					require.Empty(t, c.freight)
				}
				assert.Empty(t, recorder.Sent())
			},
		},
		{
			name:       "verifies all freight in a single request",
			freight:    []kargoapi.Freight{unverified("freight-1"), unverified("freight-2")},
			finishTime: justNow,
			assertions: func(
				t *testing.T,
				markCalls []markVerifiedCall,
				recorder *fakeevent.Sender,
				count int,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, 2, count)
				assert.Len(t, recorder.Sent(), 2)

				// Both Freight are marked verified in the Stage at the finish time.
				require.Equal(t, []markVerifiedCall{{
					project:    "default",
					stage:      "test-stage",
					freight:    []string{"freight-1", "freight-2"},
					verifiedAt: justNow,
				}}, markCalls)
			},
		},
		{
			name: "skips freight that has already been verified",
			freight: []kargoapi.Freight{
				verified("freight-1"),
				unverified("freight-2"),
			},
			finishTime: justNow,
			assertions: func(
				t *testing.T,
				markCalls []markVerifiedCall,
				recorder *fakeevent.Sender,
				count int,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, 1, count)
				require.Len(t, markCalls, 1)
				require.Equal(t, []string{"freight-2"}, markCalls[0].freight)

				require.Len(t, recorder.Sent(), 1)
				var data kargoEvent.FreightVerificationSucceeded
				require.NoError(t, recorder.Sent()[0].DataAs(&data))
				assert.Equal(t, "freight-2", data.Name)
			},
		},
		{
			name: "records event for verified freight",
			freight: []kargoapi.Freight{{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
					CreationTimestamp: metav1.Time{
						Time: oneHourAgo,
					},
				},
				Alias: "fake-alias",
			}},
			startTime:  oneMinuteAgo,
			finishTime: justNow,
			assertions: func(
				t *testing.T,
				_ []markVerifiedCall,
				recorder *fakeevent.Sender,
				_ int,
				err error,
			) {
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
			name:    "error marking freight verified",
			freight: []kargoapi.Freight{unverified("freight-1"), unverified("freight-2")},
			markErr: errors.New("something went wrong"),
			assertions: func(
				t *testing.T,
				markCalls []markVerifiedCall,
				recorder *fakeevent.Sender,
				count int,
				err error,
			) {
				require.ErrorContains(t, err, "failed to verify Freights")
				require.Zero(t, count)

				// The whole batch is attempted in one request; when it fails, no
				// Freight counts as verified and no events are sent.
				require.Len(t, markCalls, 1)
				require.Equal(t, []string{"freight-1", "freight-2"}, markCalls[0].freight)
				assert.Empty(t, recorder.Sent())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var markCalls []markVerifiedCall
			mockClient := &testutil.MockFreightClient{
				MarkFreightsVerifiedInStageFun: func(
					_ context.Context,
					project, stage string,
					freight []string,
					verifiedAt time.Time,
				) error {
					markCalls = append(
						markCalls,
						markVerifiedCall{project, stage, slices.Sorted(slices.Values(freight)), verifiedAt},
					)
					return tt.markErr
				},
			}
			recorder := &fakeevent.Sender{}

			r := &ControlFlowStageReconciler{
				eventSender:   recorder,
				freightClient: mockClient,
			}

			count, err := r.markFreightVerifiedForStage(
				t.Context(),
				testStage,
				tt.freight,
				tt.startTime,
				tt.finishTime,
			)
			tt.assertions(t, markCalls, recorder, count, err)
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
		clearErr    error
		assertions  func(*testing.T, []clearStageCall, *kargoapi.Stage, error)
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
			clearErr: errors.New("unexpected call to clear Freight status"),
			assertions: func(t *testing.T, clearCalls []clearStageCall, _ *kargoapi.Stage, err error) {
				require.NoError(t, err)
				require.Empty(t, clearCalls)
			},
		},
		{
			name: "clears Freight status in a single request and removes finalizer",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace:  "default",
					Finalizers: []string{kargoapi.FinalizerName},
				},
			},
			assertions: func(t *testing.T, clearCalls []clearStageCall, stage *kargoapi.Stage, err error) {
				require.NoError(t, err)
				assert.False(t, controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName))

				// Selecting the Freight that reference the Stage is the API
				// server's job; the reconciler asks for the Stage to be cleared once.
				require.Equal(t, []clearStageCall{{"default", "test-stage"}}, clearCalls)
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
			clearErr: errors.New("something went wrong"),
			assertions: func(t *testing.T, _ []clearStageCall, stage *kargoapi.Stage, err error) {
				require.ErrorContains(t, err, "error clearing Stage")
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
			assertions: func(t *testing.T, _ []clearStageCall, _ *kargoapi.Stage, err error) {
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

			var clearCalls []clearStageCall
			mockClient := &testutil.MockFreightClient{
				ClearStageFreightsFun: func(_ context.Context, project, stage string) error {
					clearCalls = append(clearCalls, clearStageCall{project, stage})
					return tt.clearErr
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
