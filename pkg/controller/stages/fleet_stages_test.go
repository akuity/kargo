package stages

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/indexer"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
)

func TestFleetStageReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

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
					Namespace: "default",
					Name:      "non-existent",
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "ignores control flow stage",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					// Not a regular stage
					PromotionTemplate: nil,
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "ignores regular stage",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Targets: nil,
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{}, {}},
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
			name: "shard mismatch",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
					Labels: map[string]string{
						kargoapi.LabelKeyShard: "wrong-shard",
					},
				},
				Spec: kargoapi.StageSpec{
					Shard:   "correct-shard",
					Targets: &kargoapi.StageTargets{},
					// Specify some minimal promotion process to get this Stage past the
					// logic that verifies this is not a control flow Stage.
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{}, {}},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.True(t, result.IsZero())
			},
		},
		{
			name: "handles deletion",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "default",
					Name:              "test-stage",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "deletion error",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "default",
					Name:              "test-stage",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{}, {},
							},
						},
					},
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
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{}, {},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, 100*time.Millisecond, result.RequeueAfter)

				// Verify finalizer was added
				stage := &kargoapi.Stage{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				}, stage)
				require.NoError(t, err)
				assert.Contains(t, stage.Finalizers, kargoapi.FinalizerName)
			},
		},
		{
			name: "reconcile error",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "default",
					Name:       "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{}, {},
							},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Equal(t, ctrl.Result{}, result)

				// Verify error is recorded in status
				stage := &kargoapi.Stage{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				}, stage)
				require.NoError(t, err)
			},
		},
		{
			name: "status update error after reconcile error",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "default",
					Name:       "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{}, {},
							},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
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
			name: "status update error",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "default",
					Name:       "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{}, {},
							},
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
					return fmt.Errorf("status update error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, result ctrl.Result, err error) {
				require.ErrorContains(t, err, "failed to update Stage status")
				assert.Equal(t, ctrl.Result{}, result)
			},
		},
		{
			name: "successful reconciliation",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "default",
					Name:       "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{}, {},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{RequeueAfter: 5 * time.Minute}, result)

				// Verify status was updated
				stage := &kargoapi.Stage{}
				err = c.Get(t.Context(), types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				}, stage)
				require.NoError(t, err)

				readyCond := conditions.Get(&stage.Status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := tt.objects
			if tt.stage != nil {
				objects = append(objects, tt.stage)
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(&kargoapi.Stage{}).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByWarehouseField,
					indexer.FreightByWarehouse,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByCurrentStagesField,
					indexer.FreightByCurrentStages,
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
				WithIndex(
					&kargoapi.PromotionRequest{},
					indexer.PromotionRequestsByStageField,
					indexer.PromotionRequestsByStage,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &FleetStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
			}

			result, err := r.Reconcile(t.Context(), tt.req)
			tt.assertions(t, c, result, err)
		})
	}
}

func TestFleetStagesReconciler_reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, kargoapi.StageStatus, bool, error)
	}{
		{
			name: "subreconciler error preserves reconciling condition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test-project",
					Name:       "test-stage",
					Generation: 1,
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("forced error")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, requeue bool, err error) {
				require.Error(t, err)
				require.False(t, requeue)

				reconciling := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconciling)
				assert.Equal(t, metav1.ConditionTrue, reconciling.Status)
				assert.Equal(t, "RetryAfterError", reconciling.Reason)
			},
		},
		{
			name: "intermediate status updates between subreconcilers",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test-project",
					Name:       "test-stage",
					Generation: 1,
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, requeue bool, err error) {
				require.NoError(t, err)
				require.False(t, requeue)

				// Each subreconciler should have updated conditions
				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
			},
		},
		{
			name: "clears reconciling condition when no requeue needed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test-project",
					Name:       "test-stage",
					Generation: 1,
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, requeue bool, err error) {
				require.NoError(t, err)
				assert.False(t, requeue)

				reconciling := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				assert.Nil(t, reconciling)
			},
		},
		{
			// Sub-reconcilers must see the status computed by their
			// predecessors in the same pass even when persisting that status
			// fails. Here, every Stage status update fails, yet the state
			// computed by syncPromotionRequests must survive through the rest of the
			// pass and be reflected in the returned status.
			name: "subreconcilers see status computed earlier in the pass when status updates fail",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test-project",
					Name:       "test-stage",
					Generation: 1,
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotionRequest: &kargoapi.PromotionRequestReference{Name: "test-promotion"},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "test-freight",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "test-promotion-req",
					},
					Spec: kargoapi.PromotionRequestSpec{Stage: "test-stage"},
					Status: kargoapi.PromotionRequestStatus{
						Phase:      kargoapi.PromotionRequestPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: now},
						FreightCollection: &kargoapi.FreightCollection{
							ID: "test-collection-id",
							Freight: map[string]kargoapi.FreightReference{
								"Warehouse/test-warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					ctx context.Context,
					c client.Client,
					subResourceName string,
					obj client.Object,
					patch client.Patch,
					opts ...client.SubResourcePatchOption,
				) error {
					// Fail all Stage status updates; let others through.
					if _, ok := obj.(*kargoapi.Stage); ok {
						return fmt.Errorf("status update error")
					}
					return c.SubResource(subResourceName).Patch(ctx, obj, patch, opts...)
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, requeue bool, err error) {
				// Status update failures between sub-reconcilers are non-fatal.
				require.NoError(t, err)
				require.False(t, requeue)

				// The results of syncPromotionRequests were carried through the rest
				// of the pass despite never having been persisted.
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, "test-promotion-req", status.LastPromotionRequest.Name)
				require.Len(t, status.FreightHistory, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{tt.stage}
			objects = append(objects, tt.objects...)

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByWarehouseField,
					indexer.FreightByWarehouse,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByCurrentStagesField,
					indexer.FreightByCurrentStages,
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
				WithIndex(
					&kargoapi.PromotionRequest{},
					indexer.PromotionRequestsByStageField,
					indexer.PromotionRequestsByStage,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &FleetStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
			}

			status, requeue, err := r.reconcile(t.Context(), tt.stage, now)
			tt.assertions(t, status, requeue, err)
		})
	}
}

func TestFleetStageReconciler_partitionsTargetPromotions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := metav1.NewTime(time.Now().Truncate(time.Second))
	childPromoName := api.GenerateChildPromotionName("test-stage", "blue", "test-freight")

	newChildPromo := func(phase kargoapi.PromotionPhase, finishedAt *metav1.Time) *kargoapi.Promotion {
		return &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "fake-project",
				Name:      childPromoName,
			},
			Spec: kargoapi.PromotionSpec{
				Stage:   "test-stage",
				Freight: "test-freight",
				Target:  "blue",
			},
			Status: kargoapi.PromotionStatus{
				Phase:      phase,
				FinishedAt: finishedAt,
				FreightCollection: &kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {Name: "test-freight"},
					},
				},
			},
		}
	}

	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		objects    []client.Object
		assertions func(*testing.T, kargoapi.StageStatus, bool, error)
	}{
		{
			name: "a child Promotion's success leaves the Stage's own state alone",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				newChildPromo(kargoapi.PromotionPhaseSucceeded, &now),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)

				// Nothing Stage-scoped absorbed the child's success.
				assert.Nil(t, status.CurrentPromotion)
				assert.Nil(t, status.LastPromotion)
				assert.Empty(t, status.FreightHistory)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{tt.stage.DeepCopy()}
			objects = append(objects, tt.objects...)
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.PromotionsByStageField,
					indexer.PromotionsByStage,
				).
				WithIndex(
					&kargoapi.PromotionRequest{},
					indexer.PromotionRequestsByStageField,
					indexer.PromotionRequestsByStage,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByCurrentStagesField,
					indexer.FreightByCurrentStages,
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
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Promotion{}).
				Build()

			r := &FleetStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
			}

			status, hasPendingPromotions, err := r.reconcile(t.Context(), tt.stage, time.Now())
			tt.assertions(t, status, hasPendingPromotions, err)
		})
	}
}

func TestFleetStageReconciler_syncPromotionRequests(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := metav1.NewTime(time.Now().Truncate(time.Second))
	// Generated in this order, so the ULIDs in olderRequest, newerRequest and
	// newestRequest ascend, and lex order over the three names is creation order.
	olderRequest := api.GeneratePromotionRequestName("test-stage", "test-freight")
	newerRequest := api.GeneratePromotionRequestName("test-stage", "test-freight")
	newestRequest := api.GeneratePromotionRequestName("test-stage", "test-freight")
	otherStageRequest := api.GeneratePromotionRequestName("other-stage", "test-freight")

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, kargoapi.StageStatus, error)
	}{
		{
			name: "list PromotionRequests error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("list error")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "failed to list PromotionRequests")
				assert.Nil(t, status.CurrentPromotionRequest)
				assert.Nil(t, status.LastPromotionRequest)
			},
		},
		{
			name: "no PromotionRequests clears a stale current reference",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotionRequest: &kargoapi.PromotionRequestReference{Name: olderRequest},
					LastPromotionRequest:    &kargoapi.PromotionRequestReference{Name: olderRequest},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Nil(t, status.CurrentPromotionRequest)
				// The last PromotionRequest outlives the request itself.
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, olderRequest, status.LastPromotionRequest.Name)
			},
		},
		{
			name: "PromotionRequests for other Stages are ignored",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      otherStageRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "other-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseRunning,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Nil(t, status.CurrentPromotionRequest)
				assert.Nil(t, status.LastPromotionRequest)
			},
		},
		{
			name: "non-terminal PromotionRequest becomes the current one",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseRunning,
						Freight: &kargoapi.FreightReference{
							Name: "test-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.CurrentPromotionRequest)
				assert.Equal(t, olderRequest, status.CurrentPromotionRequest.Name)
				assert.Nil(t, status.CurrentPromotionRequest.FinishedAt)
				// The reference names the Freight; it does not describe it.
				assert.Equal(
					t,
					&kargoapi.FreightReference{Name: "test-freight"},
					status.CurrentPromotionRequest.Freight,
				)
				assert.Nil(t, status.LastPromotionRequest)
			},
		},
		{
			name: "a Stage whose selectors govern no Targets still records its current request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				// An empty selector list is target-aware but selects nothing:
				// the Stage still governs Targets, it just governs none at the
				// moment. The promote endpoints create PromotionRequests for
				// such a Stage all the same, and the references mirror the
				// requests that exist, not the Stage's spec.
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{Selectors: []metav1.LabelSelector{}},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseRunning,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.CurrentPromotionRequest)
				assert.Equal(t, olderRequest, status.CurrentPromotionRequest.Name)
			},
		},
		{
			name: "a Stage that no longer governs Targets still mirrors a leftover request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				// No spec.targets at all: a Stage converted back to classic
				// with a request still in flight. The request exists, so it is
				// recorded; only its absence clears the reference.
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.CurrentPromotionRequest)
				assert.Equal(t, olderRequest, status.CurrentPromotionRequest.Name)
			},
		},
		{
			name: "a Running PromotionRequest outranks a Pending one",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhasePending,
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      newerRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseRunning,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.CurrentPromotionRequest)
				assert.Equal(t, newerRequest, status.CurrentPromotionRequest.Name)
			},
		},
		{
			name: "the older of two Pending PromotionRequests becomes the current one",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      newerRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhasePending,
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.CurrentPromotionRequest)
				assert.Equal(t, olderRequest, status.CurrentPromotionRequest.Name)
			},
		},
		{
			name: "terminal PromotionRequest becomes the last one",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotionRequest: &kargoapi.PromotionRequestReference{Name: olderRequest},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase:             kargoapi.PromotionRequestPhaseSucceeded,
						FinishedAt:        &now,
						FreightCollection: &kargoapi.FreightCollection{},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Nil(t, status.CurrentPromotionRequest)
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, olderRequest, status.LastPromotionRequest.Name)
				assert.Equal(
					t,
					kargoapi.PromotionRequestPhaseSucceeded,
					status.LastPromotionRequest.Phase,
				)
				assert.Equal(t, &now, status.LastPromotionRequest.FinishedAt)
			},
		},
		{
			name: "a terminal and a non-terminal PromotionRequest are recorded separately",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase:      kargoapi.PromotionRequestPhaseErrored,
						FinishedAt: &now,
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      newerRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseRunning,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.CurrentPromotionRequest)
				assert.Equal(t, newerRequest, status.CurrentPromotionRequest.Name)
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, olderRequest, status.LastPromotionRequest.Name)
			},
		},
		{
			name: "the last PromotionRequest never moves backwards",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					// The PromotionRequest this refers to is gone, and the one
					// still around is older than it.
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Name:  newerRequest,
						Phase: kargoapi.PromotionRequestPhaseSucceeded,
					},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      olderRequest,
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase:      kargoapi.PromotionRequestPhaseFailed,
						FinishedAt: &now,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, newerRequest, status.LastPromotionRequest.Name)
				assert.Equal(
					t,
					kargoapi.PromotionRequestPhaseSucceeded,
					status.LastPromotionRequest.Phase,
				)
			},
		},
		// Recording a succeeded PromotionRequest's Freight as the Stage's current
		// Freight. The collection is built at record time from the Freight the
		// request names and what the Stage is running then.
		{
			name: "a succeeded PromotionRequest records its Freight and resets health and verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: testOrigin("test-warehouse")}},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
					Conditions: []metav1.Condition{
						{Type: kargoapi.ConditionTypeHealthy, Status: metav1.ConditionTrue, Reason: "Healthy"},
						{Type: kargoapi.ConditionTypeVerified, Status: metav1.ConditionTrue, Reason: "Verified"},
					},
				},
			},
			objects: []client.Object{
				testFreight("test-freight", "test-warehouse"),
				testPromotionRequest(olderRequest, "test-freight", kargoapi.PromotionRequestPhaseSucceeded, &now),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.Len(t, status.FreightHistory, 1)
				assert.Equal(t, testFreightCollection("test-freight").ID, status.FreightHistory.Current().ID)
				assert.True(t, status.FreightHistory.Current().Includes("test-freight"))

				assert.Nil(t, status.Health)
				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "WaitingForHealthCheck", healthyCond.Reason)
				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionUnknown, verifiedCond.Status)
				assert.Equal(t, "WaitingForVerification", verifiedCond.Reason)
			},
		},
		{
			name: "a PromotionRequest that did not succeed records nothing",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health:         &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
					FreightHistory: kargoapi.FreightHistory{testFreightCollection("previous-freight")},
				},
			},
			objects: []client.Object{
				testFreight("test-freight", "test-warehouse"),
				testPromotionRequest(olderRequest, "test-freight", kargoapi.PromotionRequestPhaseFailed, &now),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				// The request is still the last one; the Stage's Freight is untouched.
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, olderRequest, status.LastPromotionRequest.Name)
				require.Len(t, status.FreightHistory, 1)
				assert.True(t, status.FreightHistory.Current().Includes("previous-freight"))
				assert.NotNil(t, status.Health)
			},
		},
		{
			name: "a PromotionRequest already recorded as the last one is not recorded again",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Name:  olderRequest,
						Phase: kargoapi.PromotionRequestPhaseSucceeded,
					},
					FreightHistory: kargoapi.FreightHistory{testFreightCollection("test-freight")},
					Health:         &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
				},
			},
			objects: []client.Object{
				testFreight("test-freight", "test-warehouse"),
				testPromotionRequest(olderRequest, "test-freight", kargoapi.PromotionRequestPhaseSucceeded, &now),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.Len(t, status.FreightHistory, 1)
				// Health was left alone: nothing new was recorded.
				assert.NotNil(t, status.Health)
			},
		},
		{
			name: "a newer succeeded PromotionRequest records on top of an older one",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Name:  olderRequest,
						Phase: kargoapi.PromotionRequestPhaseSucceeded,
					},
					FreightHistory: kargoapi.FreightHistory{testFreightCollection("test-freight")},
				},
			},
			objects: []client.Object{
				testFreight("test-freight", "test-warehouse"),
				testFreight("newer-freight", "test-warehouse"),
				testPromotionRequest(olderRequest, "test-freight", kargoapi.PromotionRequestPhaseSucceeded, &now),
				testPromotionRequest(newerRequest, "newer-freight", kargoapi.PromotionRequestPhaseSucceeded, &now),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, newerRequest, status.LastPromotionRequest.Name)
				require.Len(t, status.FreightHistory, 2)
				assert.True(t, status.FreightHistory[0].Includes("newer-freight"))
				assert.True(t, status.FreightHistory[1].Includes("test-freight"))
			},
		},
		{
			// Two rounds ended since the Stage last reconciled: the older request
			// succeeded and a request queued behind it failed. The newest terminal
			// request becomes the last one, but the succeeded round's Freight must
			// be recorded all the same -- examining only the newest would drop it
			// for good, since the last reference only moves forward by name.
			name: "a succeeded PromotionRequest is recorded even when a newer one failed in the same reconcile",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Name:  olderRequest,
						Phase: kargoapi.PromotionRequestPhaseSucceeded,
					},
					FreightHistory: kargoapi.FreightHistory{testFreightCollection("test-freight")},
					Health:         &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
				},
			},
			objects: []client.Object{
				testFreight("test-freight", "test-warehouse"),
				testFreight("newer-freight", "test-warehouse"),
				testFreight("newest-freight", "test-warehouse"),
				testPromotionRequest(olderRequest, "test-freight", kargoapi.PromotionRequestPhaseSucceeded, &now),
				testPromotionRequest(newerRequest, "newer-freight", kargoapi.PromotionRequestPhaseSucceeded, &now),
				testPromotionRequest(newestRequest, "newest-freight", kargoapi.PromotionRequestPhaseFailed, &now),
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				require.NotNil(t, status.LastPromotionRequest)
				assert.Equal(t, newestRequest, status.LastPromotionRequest.Name)
				assert.Equal(t, kargoapi.PromotionRequestPhaseFailed, status.LastPromotionRequest.Phase)
				require.Len(t, status.FreightHistory, 2)
				assert.True(t, status.FreightHistory[0].Includes("newer-freight"))
				assert.True(t, status.FreightHistory[1].Includes("test-freight"))
				// The succeeded round reset health.
				assert.Nil(t, status.Health)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{tt.stage.DeepCopy()}
			for _, obj := range tt.objects {
				objects = append(objects, obj.DeepCopyObject().(client.Object)) // nolint: forcetypeassert
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithIndex(
					&kargoapi.PromotionRequest{},
					indexer.PromotionRequestsByStageField,
					indexer.PromotionRequestsByStage,
				).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.PromotionRequest{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &FleetStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
			}

			status, err := r.syncPromotionRequests(t.Context(), tt.stage)
			tt.assertions(t, status, err)
		})
	}
}

func TestFleetStageReconciler_syncFreight(t *testing.T) {
	testProject := "fake-project"

	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject,
			Name:      "fake-stage",
		},
		Status: kargoapi.StageStatus{
			FreightHistory: kargoapi.FreightHistory{{
				Freight: map[string]kargoapi.FreightReference{
					"fake-warehouse-1": {Name: "fake-freight-1"},
					"fake-warehouse-2": {Name: "fake-freight-2"},
				},
			}},
		},
	}

	testCases := []struct {
		name        string
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, error)
	}{
		{
			name: "error listing Freight",
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error listing Freight in namespace")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "error getting Freight",
			interceptor: interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error getting Freight")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "successful sync",
			objects: []client.Object{
				&kargoapi.Freight{ // The Stage is using this, but the Freight doesn't know it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
				},
				&kargoapi.Freight{ // The Stage is using this, but the Freight doesn't know it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
				},
				&kargoapi.Freight{ // The Freight thinks the Stage is using this, but it's not.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-3",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // The Freight thinks the Stage is using this, but it's not.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-4",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)
				freight := &kargoapi.Freight{}
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-1"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-2"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-3"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-4"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
			},
		},
		{
			name: "removes verified freight and updates soak time when current soak is longer",
			objects: []client.Object{
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // Verified freight that should have soak time updated
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "verified-freight",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now().Add(-2 * time.Hour))), // In stage for 2 hours
							},
						},
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage.Name: {
								LongestCompletedSoak: &metav1.Duration{Duration: 30 * time.Minute}, // Previous soak was 30 min
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				// Check that expected freight remain in CurrentlyIn (they're in the stage's FreightHistory)
				freight := &kargoapi.Freight{}
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-1"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)

				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-2"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)

				// Check verified freight - should be removed from CurrentlyIn and soak time updated
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "verified-freight"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
				require.Contains(t, freight.Status.VerifiedIn, testStage.Name)
				// Soak time should be updated since 2 hours > 30 minutes
				assert.True(t, freight.Status.VerifiedIn[testStage.Name].LongestCompletedSoak.Duration > 30*time.Minute)
				assert.True(t, freight.Status.VerifiedIn[testStage.Name].LongestCompletedSoak.Duration >= time.Hour)
			},
		},
		{
			name: "removes unverified freight without affecting verification status",
			objects: []client.Object{
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // Unverified freight that should just be removed
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "unverified-freight",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now().Add(-1 * time.Hour))),
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				// Check unverified freight - should just be removed from CurrentlyIn
				freight := &kargoapi.Freight{}
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "unverified-freight"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
				require.NotContains(t, freight.Status.VerifiedIn, testStage.Name)
			},
		},
		{
			name: "preserves longer existing soak time for verified freight",
			objects: []client.Object{
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{testStage.Name: {}},
					},
				},
				&kargoapi.Freight{ // Verified freight with longer existing soak time
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "longer-soak-freight",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now().Add(-1 * time.Hour))), // In stage for 1 hour
							},
						},
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage.Name: {
								LongestCompletedSoak: &metav1.Duration{Duration: 3 * time.Hour}, // Previous soak was 3 hours
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				// Check longer soak freight - should be removed but soak time should not be updated
				freight := &kargoapi.Freight{}
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "longer-soak-freight"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
				require.Contains(t, freight.Status.VerifiedIn, testStage.Name)
				// Soak time should remain 3 hours since it's longer than the current 1 hour
				assert.Equal(t, 3*time.Hour, freight.Status.VerifiedIn[testStage.Name].LongestCompletedSoak.Duration)
			},
		},
		{
			name: "handles freight with nil Since field gracefully",
			objects: []client.Object{
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now())),
							},
						},
					},
				},
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now())),
							},
						},
					},
				},
				&kargoapi.Freight{ // Freight with nil Since field - should not panic
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "nil-since-freight",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: nil,
							},
						},
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage.Name: {
								LongestCompletedSoak: &metav1.Duration{Duration: 1 * time.Hour},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				// Check that expected freight remains in CurrentlyIn (they're in the stage's FreightHistory)
				freight := &kargoapi.Freight{}
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-1"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)

				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-2"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)

				// Should handle nil Since field gracefully without panic
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "nil-since-freight"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
				require.Contains(t, freight.Status.VerifiedIn, testStage.Name)
				// Soak time should remain unchanged as Since was nil
				assert.Equal(t, 1*time.Hour, freight.Status.VerifiedIn[testStage.Name].LongestCompletedSoak.Duration)
			},
		},
		{
			name: "handles freight with nil LongestCompletedSoak gracefully",
			objects: []client.Object{
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-1",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now())),
							},
						},
					},
				},
				&kargoapi.Freight{ // The Stage is using this, and the Freight knows it.
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "fake-freight-2",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now())),
							},
						},
					},
				},
				&kargoapi.Freight{ // Freight with nil LongestCompletedSoak
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "nil-soak-freight",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							testStage.Name: {
								Since: ptr.To(metav1.NewTime(time.Now().Add(-2 * time.Hour))),
							},
						},
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							testStage.Name: {
								LongestCompletedSoak: nil,
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				// Check that expected freight remains in CurrentlyIn (they're in the stage's FreightHistory)
				freight := &kargoapi.Freight{}
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-1"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)

				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "fake-freight-2"},
					freight,
				)
				require.NoError(t, err)
				require.Contains(t, freight.Status.CurrentlyIn, testStage.Name)

				// Should handle nil LongestCompletedSoak gracefully
				err = c.Get(
					t.Context(),
					types.NamespacedName{Namespace: testProject, Name: "nil-soak-freight"},
					freight,
				)
				require.NoError(t, err)
				require.NotContains(t, freight.Status.CurrentlyIn, testStage.Name)
				require.Contains(t, freight.Status.VerifiedIn, testStage.Name)
				// Should have created a new LongestCompletedSoak since the original was nil
				require.NotNil(t, freight.Status.VerifiedIn[testStage.Name].LongestCompletedSoak)
				assert.True(t, freight.Status.VerifiedIn[testStage.Name].LongestCompletedSoak.Duration >= time.Hour)
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testCase.objects...).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByCurrentStagesField,
					indexer.FreightByCurrentStages,
				).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				WithInterceptorFuncs(testCase.interceptor).
				Build()

			r := &FleetStageReconciler{client: c}

			err := r.syncFreight(t.Context(), testStage)
			testCase.assertions(t, c, err)
		})
	}
}

func TestFleetStageReconciler_assessHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name          string
		stage         *kargoapi.Stage
		checkHealthFn func(ctx context.Context, project, stage string, criteria []health.Criteria) kargoapi.Health
		assertions    func(*testing.T, kargoapi.StageStatus)
	}{
		{
			name: "PromotionRequest in progress",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotionRequest: &kargoapi.PromotionRequestReference{},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnknown, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "ActivePromotion", healthyCond.Reason)
				assert.Equal(t, "Stage has a PromotionRequest in progress", healthyCond.Message)
			},
		},

		{
			name: "no last promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
				},
				Status: kargoapi.StageStatus{
					LastPromotionRequest: nil,
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.Nil(t, status.Health)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "NoFreight", healthyCond.Reason)
				assert.Equal(t, "Stage has no current Freight", healthyCond.Message)
			},
		},

		{
			name: "no healthchecks on fleet stages",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Targets: &kargoapi.StageTargets{},
				},
				Status: kargoapi.StageStatus{
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Phase: kargoapi.PromotionRequestPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateHealthy, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionTrue, healthyCond.Status)
				assert.Equal(t, "TargetAwareStage", healthyCond.Reason)
				assert.Equal(t, "Health is assessed per Target, not for the Stage", healthyCond.Message)
			},
		},

		{
			name: "unsuccessful last promotion request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Phase: kargoapi.PromotionRequestPhaseAborted,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnknown, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "LastPromotionRequestAborted", healthyCond.Reason)
				assert.Equal(t, "Cannot assess health because last PromotionRequest did not succeed", healthyCond.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			r := &FleetStageReconciler{
				client: c,
			}

			status := r.assessHealth(t.Context(), tt.stage)
			tt.assertions(t, status)
		})
	}
}

func TestFleetStageReconciler_verifyStageFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

	tests := []struct {
		name             string
		stage            *kargoapi.Stage
		objects          []client.Object
		assertions       func(*testing.T, client.Client, *fakeevent.EventRecorder, kargoapi.StageStatus, error)
		rolloutsDisabled bool
	}{
		{
			name: "no current freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: nil,
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionUnknown, verifiedCond.Status)
				assert.Equal(t, "NoFreight", verifiedCond.Reason)
			},
		},
		{
			name: "skips verification when promotion is running",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotionRequest: &kargoapi.PromotionRequestReference{
						Name: "running-promotion",
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				assert.Nil(t, verifiedCond)
			},
		},
		{
			name: "verifies without verification config",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: nil,
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse":   {Name: "test-freight"},
								"warehouse-2": {Name: "test-freight-2"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight-2",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, recorder.Events, 2)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, lastVerification.Phase)
				assert.Equal(t, metav1.NewTime(startTime), *lastVerification.StartTime)
				assert.Equal(t, metav1.NewTime(endTime), *lastVerification.FinishTime)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)
			},
		},
		{
			name: "skips verification with nil health status",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: nil,
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				_ *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)
				assert.Empty(t, curFreight.VerificationHistory)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.Nil(t, verifiedCond)
			},
		},
		{
			name: "skips verification when unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				_ *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)
				assert.Empty(t, curFreight.VerificationHistory)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.Nil(t, verifiedCond)
			},
		},
		{
			name:             "error when rollouts integration is disabled",
			rolloutsDisabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseError, lastVerification.Phase)
				assert.Contains(t, lastVerification.Message, "Rollouts integration is disabled")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationError", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "Rollouts integration is disabled")
			},
		},
		{
			name: "handles verification abort request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAbort: "test-verification-id",
					},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseRunning,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "test-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis-run",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				c client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, lastVerification.Phase)
				assert.Contains(t, lastVerification.Message, "aborted by user")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationFailed", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "aborted by user")

				// Verify AnalysisRun was patched to terminate
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Name:      "test-analysis-run",
					Namespace: "fake-project",
				}, ar))
				assert.True(t, ar.Spec.Terminate)
			},
		},
		{
			name: "handles re-verification request",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: `{"id":"test-verification-id","actor":"test-user"}`,
					},
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					LastPromotionRequest: &kargoapi.PromotionRequestReference{
						Name: "last-promotion",
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				c client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhasePending, lastVerification.Phase)
				assert.NotEmpty(t, lastVerification.ID)
				assert.Equal(t, "test-user", lastVerification.Actor)

				// As we have a successful (previous) verification, we should have a verified condition
				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)

				// Verify new AnalysisRun was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Name:      lastVerification.AnalysisRun.Name,
					Namespace: lastVerification.AnalysisRun.Namespace,
				}, ar))
			},
		},
		{
			name: "continues existing non-terminal verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:        "test-verification-id",
									Phase:     kargoapi.VerificationPhaseRunning,
									StartTime: &metav1.Time{Time: startTime},
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "test-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis-run",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Running",
						Message: "Analysis is running",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 0)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseRunning, lastVerification.Phase)
				assert.Equal(t, "test-verification-id", lastVerification.ID)
				assert.Equal(t, "test-analysis-run", lastVerification.AnalysisRun.Name)
				assert.Equal(t, "Running", lastVerification.AnalysisRun.Phase)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionUnknown, verifiedCond.Status)
				assert.Equal(t, "VerificationRunning", verifiedCond.Reason)
				assert.Equal(t, "Freight is currently being verified", verifiedCond.Message)
			},
		},
		{
			name: "handles error getting AnalysisRun for freight verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseRunning,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "missing-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.True(t, apierrors.IsNotFound(err))

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseError, lastVerification.Phase)
				assert.Contains(t, lastVerification.Message, "error getting AnalysisRun")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationError", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "error getting AnalysisRun")
			},
		},
		{
			name: "uses existing analysis run for freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-freight-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed successfully",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, lastVerification.Phase)
				assert.Equal(t, "existing-analysis", lastVerification.AnalysisRun.Name)

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)
			},
		},
		{
			name: "handles multiple verification histories with re-verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyReverify: `{"id":"second-verification","actor":"test-user"}`,
					},
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:        "second-verification",
									Phase:     kargoapi.VerificationPhaseSuccessful,
									StartTime: &metav1.Time{Time: startTime.Add(-time.Hour)},
								},
								{
									ID:        "first-verification",
									Phase:     kargoapi.VerificationPhaseSuccessful,
									StartTime: &metav1.Time{Time: startTime.Add(-2 * time.Hour)},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				_ *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)
				require.Len(t, curFreight.VerificationHistory, 3)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhasePending, lastVerification.Phase)
				assert.NotEmpty(t, lastVerification.ID)
				assert.Equal(t, "test-user", lastVerification.Actor)

				// Should be true as we have a successful verification
				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionTrue, verifiedCond.Status)
				assert.Equal(t, "Verified", verifiedCond.Reason)
				assert.Equal(t, "Freight has been verified", verifiedCond.Message)

				// Verify the previous verifications are preserved
				assert.Equal(t, "second-verification", curFreight.VerificationHistory[1].ID)
				assert.Equal(t, "first-verification", curFreight.VerificationHistory[2].ID)
			},
		},
		{
			name: "handles terminal analysis run state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-freight-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									ID:    "test-verification-id",
									Phase: kargoapi.VerificationPhaseRunning,
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Name:      "test-analysis-run",
										Namespace: "fake-project",
									},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis-run",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Failed",
						Message: "Analysis failed due to metric error",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ client.Client,
				recorder *fakeevent.EventRecorder,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				require.Len(t, recorder.Events, 1)

				curFreight := status.FreightHistory.Current()
				require.NotNil(t, curFreight)

				lastVerification := curFreight.VerificationHistory.Current()
				require.NotNil(t, lastVerification)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, lastVerification.Phase)
				assert.Equal(t, "test-analysis-run", lastVerification.AnalysisRun.Name)
				assert.Equal(t, "Failed", lastVerification.AnalysisRun.Phase)
				assert.Contains(t, lastVerification.Message, "Analysis failed")

				verifiedCond := conditions.Get(&status, kargoapi.ConditionTypeVerified)
				require.NotNil(t, verifiedCond)
				assert.Equal(t, metav1.ConditionFalse, verifiedCond.Status)
				assert.Equal(t, "VerificationFailed", verifiedCond.Reason)
				assert.Contains(t, verifiedCond.Message, "Analysis failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}).
				Build()

			recorder := fakeevent.NewEventRecorder(10)

			r := &FleetStageReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				},
				eventSender: k8sevent.NewEventSender(recorder),
				backoffCfg: wait.Backoff{
					Duration: 1 * time.Second,
					Factor:   2,
					Steps:    2,
					Cap:      2 * time.Second,
					Jitter:   0.1,
				},
			}

			status, err := r.verifyStageFreight(t.Context(), tt.stage, startTime, fixedEndTime)
			tt.assertions(t, c, recorder, status, err)
		})
	}
}

func TestFleetStageReconciler_markFreightVerifiedForStage(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	endTime := metav1.Now()

	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		objects    []client.Object
		assertions func(*testing.T, client.Client, kargoapi.StageStatus, error)
	}{
		{
			name: "skips verification when unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnhealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				// Status should remain unchanged
				assert.Equal(t, kargoapi.HealthStateUnhealthy, status.Health.Status)
			},
		},
		{
			name: "skips verification when no current freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "skips verification when non-terminal verification exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseRunning,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "skips verification when last verification is not successful",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseFailed,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "handles freight not found",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "missing-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error getting Freight")
			},
		},
		{
			name: "marks freight as verified when not already verified",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase:      kargoapi.VerificationPhaseSuccessful,
									FinishTime: endTime.DeepCopy(),
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-freight",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				// Check if freight was properly marked as verified
				freight := &kargoapi.Freight{}
				require.NoError(t, c.Get(t.Context(), client.ObjectKey{
					Namespace: "fake-project",
					Name:      "test-freight",
				}, freight))

				verifiedStage, ok := freight.Status.VerifiedIn["test-stage"]
				require.True(t, ok)
				assert.Equal(t, endTime.Unix(), verifiedStage.VerifiedAt.Unix())
			},
		},
		{
			name: "skips already verified freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-freight",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				// Verify no changes were made to the freight
				freight := &kargoapi.Freight{}
				require.NoError(t, c.Get(t.Context(), client.ObjectKey{
					Namespace: "fake-project",
					Name:      "test-freight",
				}, freight))
				assert.Len(t, freight.Status.VerifiedIn, 1)
			},
		},
		{
			name: "handles multiple freight references",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse1": {Name: "freight-1"},
								"warehouse2": {Name: "freight-2"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase:      kargoapi.VerificationPhaseSuccessful,
									FinishTime: ptr.To(endTime.Rfc3339Copy()),
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "freight-1",
					},
					Status: kargoapi.FreightStatus{},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "freight-2",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			assertions: func(t *testing.T, c client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)

				// Check both freight objects were marked as verified
				for _, name := range []string{"freight-1", "freight-2"} {
					freight := &kargoapi.Freight{}
					require.NoError(t, c.Get(t.Context(), client.ObjectKey{
						Namespace: "fake-project",
						Name:      name,
					}, freight))

					verifiedStage, ok := freight.Status.VerifiedIn["test-stage"]
					require.True(t, ok, "freight %s should be verified", name)
					assert.Equal(t, endTime.Unix(), verifiedStage.VerifiedAt.Unix())
				}
			},
		},
		{
			name: "handles patch error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase:      kargoapi.VerificationPhaseSuccessful,
									FinishTime: &metav1.Time{Time: endTime.Time},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       "fake-project",
						Name:            "test-freight",
						ResourceVersion: "invalid", // This will cause patch to fail
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "error marking Freight")
			},
		},
		{
			name: "empty verification history skips verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateHealthy,
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "nil health status skips verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: nil,
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "test-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "test-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase: kargoapi.VerificationPhaseSuccessful,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, _ kargoapi.StageStatus, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				Build()

			r := &FleetStageReconciler{
				client: c,
			}

			status, err := r.markFreightVerifiedForStage(t.Context(), tt.stage)
			tt.assertions(t, c, status, err)
		})
	}
}

func TestFleetStageReconciler_autoPromoteFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	tests := []struct {
		name                 string
		autoPromotionEnabled bool
		stage                *kargoapi.Stage
		objects              []client.Object
		interceptor          interceptor.Funcs
		assertions           func(*testing.T, *fakeevent.EventRecorder, client.Client, kargoapi.StageStatus, error)
	}{
		{
			name:                 "no requested freight",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: nil,
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name:                 "auto-promotion not allowed",
			autoPromotionEnabled: false,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.False(t, status.AutoPromotionEnabled)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name:                 "handles direct freight from warehouse",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-2",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// Verify promotion was created for newest freight
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name:                 "sorts by discoveredAt when set",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{Uses: "fake-step"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				// freight-1 has an older creationTimestamp but a newer discoveredAt
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					DiscoveredAt: &metav1.Time{Time: now},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				// freight-2 has a newer creationTimestamp but an older discoveredAt
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-2",
						CreationTimestamp: metav1.Time{Time: now},
					},
					DiscoveredAt: &metav1.Time{Time: hourAgo},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// freight-1 wins because it has the newer discoveredAt,
				// even though freight-2 has a newer creationTimestamp.
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name:                 "falls back to creationTimestamp when discoveredAt is unset",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{Uses: "fake-step"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-2",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// freight-1 wins because it has the newer creationTimestamp
				// (neither has a discoveredAt set).
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name:                 "skips promotion when current freight is latest",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"Warehouse/test-warehouse": {Name: "test-freight-1"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name:                 "skips promotion if a non-terminal one already exists",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// Verify no new promotions were created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			// A fast Promotion can go from pending to succeeded in the interval
			// between syncPromotions observing it and autoPromoteFreight acting.
			// Its outcome is not recorded yet (it is newer than
			// status.lastPromotionRequest), so auto-promotion must stand down rather
			// than create a duplicate.
			name:                 "skips promotion when a succeeded one for the candidate is not yet recorded",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
				// No LastPromotionRequest: the succeeded Promotion below has not been
				// processed by syncPromotions.
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseSucceeded,
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// Verify no duplicate promotion was created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			name:                 "skips promotion if the last terminal one was not successful",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.PromotionRequest{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionRequestSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
					Status: kargoapi.PromotionRequestStatus{
						Phase: kargoapi.PromotionRequestPhaseErrored,
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// Verify no new promotions were created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			name:                 "handles verified freight from upstream stages",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"upstream-stage"},
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				assert.True(t, status.AutoPromotionEnabled)

				// Verify promotion was created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name:                 "handles verified freight from upstream stages with soak time requirement",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages:           []string{"upstream-stage"},
								RequiredSoakTime: &metav1.Duration{Duration: time.Hour},
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							// Ignored because it does not have a timestamp.
							"upstream-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-2",
						CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {
								// Should be selected because the soak time has elapsed
								LongestCompletedSoak: &metav1.Duration{Duration: 2 * time.Hour},
							},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-3",
						CreationTimestamp: metav1.Time{Time: now.Add(-40 * time.Minute)},
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {
								// Should be ignored because it is too recent.
								VerifiedAt: &metav1.Time{Time: now.Add(-39 * time.Minute)},
							},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotion was created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-2", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name:                 "handles freight approved for stage",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotion was created
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name:                 "handles multiple freight requests",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-2",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "warehouse-1",
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "warehouse-2",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-1",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-2",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-2",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify promotions were created for both freight items
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 2)

				// Verify they're for different freight
				freightNames := map[string]bool{}
				for _, promo := range promoList.Items {
					freightNames[promo.Spec.Freight] = true
				}
				assert.Len(t, freightNames, 2)
				assert.True(t, freightNames["freight-1"])
				assert.True(t, freightNames["freight-2"])
			},
		},
		{
			name:                 "deduplicates freight from multiple sources",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Stages: []string{"upstream-stage"},
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
						ApprovedFor: map[string]kargoapi.ApprovedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// Verify only one promotion was created despite multiple sources
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
			},
		},
		{
			name:                 "handles promotion creation error",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{},
				},
			},
			interceptor: interceptor.Funcs{
				Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				_ client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.ErrorContains(t, err, "error creating PromotionRequest")
			},
		},
		{
			// A Forbidden error from an admission webhook must not fail the
			// reconcile: nothing is persisted and the pass continues, so a later
			// reconcile can re-attempt once the denying policy no longer applies.
			name:                 "tolerates a forbidden PromotionRequest create",
			autoPromotionEnabled: true,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
			},
			interceptor: interceptor.Funcs{
				Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
					return &apierrors.StatusError{ErrStatus: metav1.Status{
						Status: metav1.StatusFailure,
						Reason: metav1.StatusReasonForbidden,
						// Shaped like a real denial: the API server's wrapper around
						// what a webhook returned via apierrors.NewForbidden, naming
						// the Promotion the mutating webhook had just generated.
						Message: `admission webhook "fake-webhook" denied the request: ` +
							`promotions.kargo.akuity.io "test-stage.01jzzz.abc1234" is forbidden: ` +
							`promotion of Stage "test-stage" is not permitted at this time`,
					}}
				},
			},
			assertions: func(
				t *testing.T,
				recorder *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)

				// No Promotion is persisted when the create is denied.
				promoList := &kargoapi.PromotionRequestList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)

				// Nor is anything recorded. A denial is a condition that persists
				// across reconciles, not an occurrence, so an event per attempt
				// would report the same thing indefinitely.
				assert.Empty(t, recorder.Events)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := append([]client.Object{tt.stage}, tt.objects...)
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
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
					indexer.FreightByCurrentStagesField,
					indexer.FreightByCurrentStages,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesField,
					indexer.FreightApprovedForStages,
				).
				WithIndex(
					&kargoapi.PromotionRequest{},
					indexer.PromotionRequestsByStageAndFreightField,
					indexer.PromotionRequestsByStageAndFreight,
				)

			c := builder.Build()
			recorder := fakeevent.NewEventRecorder(5)

			r := &FleetStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(recorder),
			}

			status, err := r.autoPromoteFreight(t.Context(), tt.stage, tt.autoPromotionEnabled)
			tt.assertions(t, recorder, c, status, err)
		})
	}
}

func Test_summarizeFleetConditions(t *testing.T) {
	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		status     *kargoapi.StageStatus
		err        error
		assertions func(*testing.T, *kargoapi.StageStatus, bool)
	}{
		{
			name: "with error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{},
			err:    errors.New("something went wrong"),
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "ReconcileError", readyCond.Reason)
				assert.Equal(t, "something went wrong", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				reconcileCond := conditions.Get(status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcileCond)
				assert.Equal(t, metav1.ConditionTrue, reconcileCond.Status)
				assert.Equal(t, "RetryAfterError", reconcileCond.Reason)
				assert.Equal(t, int64(1), reconcileCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "promoting",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypePromoting,
						Status:  metav1.ConditionTrue,
						Reason:  "Promoting",
						Message: "Stage is promoting",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "Promoting", readyCond.Reason)
				assert.Equal(t, "Stage is promoting", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "last promotion failed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				LastPromotionRequest: &kargoapi.PromotionRequestReference{
					Phase:   kargoapi.PromotionRequestPhaseFailed,
					Message: "Promotion failed due to error",
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "LastPromotionRequestFailed", readyCond.Reason)
				assert.Equal(t, "Promotion failed due to error", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "unhealthy",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionFalse,
						Reason:  "HealthCheckFailed",
						Message: "Health check failed",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)

				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "HealthCheckFailed", readyCond.Reason)
				assert.Equal(t, "Health check failed", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "missing health condition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "Unhealthy", readyCond.Reason)
				assert.Equal(t, "Stage is not healthy", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "health unknown",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionUnknown,
						Reason:  "HealthCheckPending",
						Message: "Health check in progress",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "HealthCheckPending", readyCond.Reason)
				assert.Equal(t, "Health check in progress", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "pending verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionUnknown,
						Reason:  "VerificationPending",
						Message: "Verification is pending",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "VerificationPending", readyCond.Reason)
				assert.Equal(t, "Verification is pending", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "verification error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionFalse,
						Reason:  "VerificationError",
						Message: "Verification failed",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "VerificationError", readyCond.Reason)
				assert.Equal(t, "Verification failed", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
				assert.False(t, ready)
			},
		},
		{
			name: "missing verification condition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "PendingVerification", readyCond.Reason)
				assert.Equal(t, "Stage is not verified", readyCond.Message)
				assert.False(t, ready)
			},
		},
		{
			name: "ready",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionTrue,
						Reason:  "Verified",
						Message: "Stage is verified",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)

				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
				assert.Equal(t, "Verified", readyCond.Reason)
				assert.Equal(t, "Stage is verified", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.True(t, ready)
			},
		},
		{
			name: "reconciling condition cleared when ready",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			status: &kargoapi.StageStatus{
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionTrue,
						Reason:  "Verified",
						Message: "Stage is verified",
					},
					{
						Type:    kargoapi.ConditionTypeReconciling,
						Status:  metav1.ConditionTrue,
						Reason:  "Reconciling",
						Message: "Stage is reconciling",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)

				reconcileCond := conditions.Get(status, kargoapi.ConditionTypeReconciling)
				assert.Nil(t, reconcileCond, "Reconciling condition should be deleted when ready")

				assert.True(t, ready)
			},
		},
		{
			name: "freight summary updated",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{}, {}},
				},
			},
			status: &kargoapi.StageStatus{
				FreightHistory: kargoapi.FreightHistory{
					&kargoapi.FreightCollection{
						Freight: map[string]kargoapi.FreightReference{
							"freight1": {Name: "freight1"},
						},
					},
				},
				Conditions: []metav1.Condition{
					{
						Type:    kargoapi.ConditionTypeHealthy,
						Status:  metav1.ConditionTrue,
						Reason:  "Healthy",
						Message: "Stage is healthy",
					},
					{
						Type:    kargoapi.ConditionTypeVerified,
						Status:  metav1.ConditionTrue,
						Reason:  "Verified",
						Message: "Stage is verified",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus, ready bool) {
				assert.Equal(t, "1/2 Fulfilled", status.FreightSummary)
				assert.True(t, ready)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready := summarizeFleetConditions(tt.stage, tt.status, tt.err)
			tt.assertions(t, tt.status, ready)
		})
	}
}

// testFreightCollection returns the FreightCollection a PromotionRequest for
// the named Freight would carry, built the way the request reconciler builds
// it, so that its ID is the real one.
func testFreightCollection(freight string) *kargoapi.FreightCollection {
	collection := &kargoapi.FreightCollection{}
	collection.UpdateOrPush(kargoapi.FreightReference{
		Name: freight,
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "test-warehouse",
		},
	})
	return collection
}

// testPromotionRequest returns a PromotionRequest of the test Stage for the
// named Freight, in the given phase.
func testPromotionRequest(
	name string,
	freight string,
	phase kargoapi.PromotionRequestPhase,
	finishedAt *metav1.Time,
) *kargoapi.PromotionRequest {
	freightRef := kargoapi.FreightReference{
		Name: freight,
	}
	freightCollection := &kargoapi.FreightCollection{}
	freightCollection.UpdateOrPush(freightRef)

	return &kargoapi.PromotionRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fake-project",
			Name:      name,
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   "test-stage",
			Freight: freight,
		},
		Status: kargoapi.PromotionRequestStatus{
			Phase:             phase,
			FinishedAt:        finishedAt,
			Freight:           &freightRef,
			FreightCollection: freightCollection,
		},
	}
}
