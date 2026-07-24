package stages

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/akuity/kargo/pkg/credentials"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/indexer"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
)

func TestRegularStageReconciler_Reconcile(t *testing.T) {
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
					Shard: "correct-shard",
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
			name: "status update error after syncing Promotions",
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
					&kargoapi.Promotion{},
					indexer.PromotionsByStageField,
					indexer.PromotionsByStage,
				).
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
					&kargoapi.Promotion{},
					indexer.PromotionsByStageAndFreightField,
					indexer.PromotionsByStageAndFreight,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &RegularStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
			}

			result, err := r.Reconcile(t.Context(), tt.req)
			tt.assertions(t, c, result, err)
		})
	}
}

func TestRegularStagesReconciler_reconcile(t *testing.T) {
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
			// computed by syncPromotions must survive through the rest of the
			// pass and be reflected in the returned status.
			name: "subreconcilers see status computed earlier in the pass when status updates fail",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test-project",
					Name:       "test-stage",
					Generation: 1,
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{Name: "test-promotion"},
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
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-project",
						Name:      "test-promotion",
					},
					Spec: kargoapi.PromotionSpec{Stage: "test-stage"},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
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

				// The results of syncPromotions were carried through the rest
				// of the pass despite never having been persisted.
				require.NotNil(t, status.LastPromotion)
				assert.Equal(t, "test-promotion", status.LastPromotion.Name)
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
					&kargoapi.Promotion{},
					indexer.PromotionsByStageField,
					indexer.PromotionsByStage,
				).
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
					&kargoapi.Promotion{},
					indexer.PromotionsByStageAndFreightField,
					indexer.PromotionsByStageAndFreight,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &RegularStageReconciler{
				client:        c,
				eventSender:   k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
				healthChecker: &health.MockAggregatingChecker{},
			}

			status, requeue, err := r.reconcile(t.Context(), tt.stage, now)
			tt.assertions(t, status, requeue, err)
		})
	}
}

// releaseHoldAnnotations returns the annotations set on a release-intent Promotion.
func releaseHoldAnnotations(origin kargoapi.FreightOrigin) map[string]string {
	promo := &kargoapi.Promotion{}
	api.SetAutoPromotionResumeAnnotation(promo, origin)
	return promo.Annotations
}

func TestRegularStageReconciler_syncPromotions(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := time.Now().Truncate(time.Second)
	hourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)
	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "test-warehouse",
	}

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, kargoapi.StageStatus, bool, error)
	}{
		{
			name: "list promotions error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("list error")
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.ErrorContains(t, err, "failed to list Promotions")
				assert.False(t, hasPendingPromotions)

				assert.Len(t, status.Conditions, 1)
				assert.Equal(t, kargoapi.ConditionTypePromoting, status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionUnknown, status.Conditions[0].Status)
				require.Contains(t, status.Conditions[0].Message, "failed to list Promotions")
			},
		},
		{
			name: "no promotions",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "old-promotion",
					},
					Conditions: []metav1.Condition{
						{
							Type: kargoapi.ConditionTypePromoting,
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)
				assert.Empty(t, status.Conditions)
			},
		},
		{
			name: "hold-intent promotion creates hold",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{Name: "hold-promo"},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "hold-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: origin.String(),
							kargoapi.AnnotationKeyCreateActor:       "user:alice",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "older-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				require.Len(t, status.AutoPromotionHolds, 1)
				hold := status.AutoPromotionHolds[origin.String()]
				assert.Equal(t, "older-freight", hold.FreightName)
				assert.Equal(t, "hold-promo", hold.PromotionName)
				assert.Equal(t, "user:alice", hold.Actor)
				assert.True(t, hold.Origin.Equals(&origin))
			},
		},
		{
			name: "hold-intent promotion older than release yields no hold",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{Name: "release-promo"},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "hold-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: origin.String(),
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "old-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "release-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations:       releaseHoldAnnotations(origin),
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "new-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Empty(t, status.AutoPromotionHolds)
			},
		},
		{
			name: "hold and release at same timestamp use name ordering",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{Name: "b-release-promo"},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "a-hold-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: origin.String(),
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "old-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "b-release-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations:       releaseHoldAnnotations(origin),
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "new-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Empty(t, status.AutoPromotionHolds)
			},
		},
		{
			name: "hold-intent newer than release creates hold",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{Name: "b-hold-promo"},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "a-release-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Annotations:       releaseHoldAnnotations(origin),
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "mid-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "b-hold-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: origin.String(),
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "newer-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				require.Len(t, status.AutoPromotionHolds, 1)
				hold := status.AutoPromotionHolds[origin.String()]
				assert.Equal(t, "b-hold-promo", hold.PromotionName)
			},
		},
		{
			name: "hold-intent for unrequested origin is ignored",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{Name: "hold-promo"},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hold-promo",
						Namespace: "fake-project",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/other-warehouse",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "some-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Empty(t, status.AutoPromotionHolds)
			},
		},
		{
			name: "existing hold survives promotion garbage collection",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						origin.String(): {
							FreightName:   "older-freight",
							Origin:        origin,
							PromotionName: "garbage-collected-promotion",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				require.Len(t, status.AutoPromotionHolds, 1)
				assert.Equal(
					t,
					"garbage-collected-promotion",
					status.AutoPromotionHolds[origin.String()].PromotionName,
				)
			},
		},
		{
			// The auto-promotion candidate rotated while a hold was active.
			// The user promotes the new candidate (not the original hold Freight).
			// The hold must clear because the resume annotation — not the Freight
			// name — is the authoritative signal.
			name: "hold is cleared when rotated candidate is promoted",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						origin.String(): {
							FreightName:   "freight-original",
							Origin:        origin,
							PromotionName: "original-hold-promo",
						},
					},
					CurrentPromotion: &kargoapi.PromotionReference{Name: "rotated-candidate-promo"},
				},
			},
			objects: []client.Object{
				// The original hold-intent Promotion (older).
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "original-hold-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: origin.String(),
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-original",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
				// The user later promoted the new candidate (different Freight),
				// stamping a resume annotation. This should clear the hold.
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "rotated-candidate-promo",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations:       releaseHoldAnnotations(origin),
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-rotated-new",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Empty(
					t,
					status.AutoPromotionHolds,
					"resume annotation on the rotated-candidate Promotion should clear the hold",
				)
			},
		},
		{
			// Two origins; in the same reconcile pass one origin's hold is
			// cleared by a resume Promotion and the other gains a new hold from
			// a hold-intent Promotion.
			name: "hold cleared for one origin and established for another in same reconcile",
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
								Name: "warehouse-a",
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-b",
							},
						},
					},
				},
				Status: kargoapi.StageStatus{
					// Origin A has an active hold; origin B does not.
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/warehouse-a": {
							FreightName: "freight-a-old",
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-a",
							},
							PromotionName: "hold-promo-a",
						},
					},
					CurrentPromotion: &kargoapi.PromotionReference{Name: "hold-promo-b"},
				},
			},
			objects: []client.Object{
				// Origin A: release Promotion (newer) → hold must clear.
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "hold-promo-a",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/warehouse-a",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-a-old",
					},
					Status: kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "release-promo-a",
						Namespace: "fake-project",
						// Newer than hold-promo-a → outranks it for origin A.
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Annotations: releaseHoldAnnotations(kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "warehouse-a",
						}),
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-a-new",
					},
					Status: kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded},
				},
				// Origin B: hold-intent Promotion (newest overall) → new hold.
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hold-promo-b",
						Namespace: "fake-project",
						// Newer than release-promo-a → wins for origin B.
						CreationTimestamp: metav1.Time{Time: now},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/warehouse-b",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-b-old",
					},
					Status: kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)

				// Origin A's hold must be cleared.
				_, hasHoldA := status.AutoPromotionHolds["Warehouse/warehouse-a"]
				assert.False(t, hasHoldA, "origin A hold should be cleared by its resume Promotion")

				// Origin B must have a new hold.
				holdB, hasHoldB := status.AutoPromotionHolds["Warehouse/warehouse-b"]
				assert.True(t, hasHoldB, "origin B should have a new hold from its hold-intent Promotion")
				assert.Equal(t, "freight-b-old", holdB.FreightName)
			},
		},
		{
			name: "existing hold is removed when origin is no longer requested",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "other-warehouse",
						},
					}},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						origin.String(): {
							FreightName:   "older-freight",
							Origin:        origin,
							PromotionName: "hold-promotion",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Empty(t, status.AutoPromotionHolds)
			},
		},
		{
			// The Stage controller does NOT abort in-flight Promotions when a hold
			// exists; it simply refuses to create new ones (in autoPromoteFreight).
			name: "in-flight auto-promotion is not aborted when hold exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/test-warehouse": {
							FreightName:   "older-freight",
							Origin:        origin,
							PromotionName: "hold-promo",
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "newest-freight",
						Namespace: "fake-project",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "auto-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "newest-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hold-promo",
						Namespace: "fake-project",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: origin.String(),
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "older-freight",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "auto-promotion", status.CurrentPromotion.Name)
				require.Len(t, status.AutoPromotionHolds, 1)
			},
		},
		{
			name: "successful promotion updates freight history",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{
						{
							Type: kargoapi.ConditionTypePromoting,
						},
					},
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "successful-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "successful-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: now},
						FreightCollection: &kargoapi.FreightCollection{
							ID: "test-collection-id",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {
									Name: "test-freight-1",
								},
								"warehouse-2": {
									Name: "test-freight-2",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)

				assert.False(t, hasPendingPromotions)

				assert.Nil(t, status.CurrentPromotion)

				require.NotNil(t, status.LastPromotion)
				assert.Equal(t, "successful-promotion", status.LastPromotion.Name)

				// Verify freight history
				require.Len(t, status.FreightHistory, 1)
				assert.Equal(t, &kargoapi.FreightCollection{
					ID: "test-collection-id",
					Freight: map[string]kargoapi.FreightReference{
						"warehouse-1": {
							Name: "test-freight-1",
						},
						"warehouse-2": {
							Name: "test-freight-2",
						},
					},
				}, status.FreightHistory[0])

				// Verify conditions are set correctly
				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)

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
			name: "active promotion updates status",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "active-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
						Freight: &kargoapi.FreightReference{
							Name: "test-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)

				assert.True(t, hasPendingPromotions)

				assert.Equal(t, &kargoapi.PromotionReference{
					Name: "active-promotion",
					Freight: &kargoapi.FreightReference{
						Name: "test-freight",
					},
				}, status.CurrentPromotion)

				// Verify promoting condition is set
				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
			},
		},
		{
			name: "blocks new promotion when current freight has running verification",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {
									Name: "current-freight",
								},
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
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)
				assert.Empty(t, status.Conditions)
			},
		},
		{
			name: "waits for running verification even when health is unhealthy",
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
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{Phase: kargoapi.VerificationPhaseRunning},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should not have current promotion since verification is running
				assert.Nil(t, status.CurrentPromotion)
				// Should not have promoting condition
				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
		{
			name: "allows promotion after verification completes when health is unhealthy",
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
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{
									Phase:      kargoapi.VerificationPhaseSuccessful,
									FinishTime: &metav1.Time{Time: time.Now()},
								},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should allow promotion after verification is complete
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "pending-promotion", status.CurrentPromotion.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
			},
		},
		{
			name: "waits for verification even with no health check",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: nil, // No health check performed
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							VerificationHistory: []kargoapi.VerificationInfo{
								{Phase: kargoapi.VerificationPhaseRunning},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
		{
			name: "allows promotion when unhealthy and no verification exists",
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
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
							},
							// Empty verification history
							VerificationHistory: []kargoapi.VerificationInfo{},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
						Freight: &kargoapi.FreightReference{
							Name: "new-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should allow promotion since there's no verification to wait for
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "pending-promotion", status.CurrentPromotion.Name)
				assert.Equal(t, "new-freight", status.CurrentPromotion.Freight.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
				assert.Contains(t, promotingCond.Message, "Pending")
			},
		},
		{
			name: "allows promotion when health is unknown and no verification exists",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Health: &kargoapi.Health{
						Status: kargoapi.HealthStateUnknown,
						Issues: []string{"Cannot assess health because last Promotion did not succeed"},
					},
					FreightHistory: kargoapi.FreightHistory{
						{
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse": {Name: "current-freight"},
							},
							// No verification history
							VerificationHistory: []kargoapi.VerificationInfo{},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
						Freight: &kargoapi.FreightReference{
							Name: "new-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should allow promotion since there's no verification to wait for
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "pending-promotion", status.CurrentPromotion.Name)
				assert.Equal(t, "new-freight", status.CurrentPromotion.Freight.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
				assert.Contains(t, promotingCond.Message, "Pending")
			},
		},
		{
			name: "allows promotion when verification failed",
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
							ID: "current-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "current-freight"},
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
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pending-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
						Freight: &kargoapi.FreightReference{
							Name: "new-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				// Should allow promotion since there's no verification to wait for
				require.NotNil(t, status.CurrentPromotion)
				assert.Equal(t, "pending-promotion", status.CurrentPromotion.Name)
				assert.Equal(t, "new-freight", status.CurrentPromotion.Freight.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
				assert.Contains(t, promotingCond.Message, "Pending")
			},
		},
		{
			name: "skips older promotions after last promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "promotion-2",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "promotion-1",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: hourAgo},
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "promotion-2",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: now},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)

				assert.Equal(t, "promotion-2", status.LastPromotion.Name)
				assert.Empty(t, status.FreightHistory)
			},
		},
		{
			name: "processes failed promotions without updating freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "failed-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "failed-promotion",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseFailed,
						FinishedAt: &metav1.Time{Time: now},
						FreightCollection: &kargoapi.FreightCollection{
							ID: "failed-collection",
							Freight: map[string]kargoapi.FreightReference{
								"warehouse-1": {Name: "failed-freight"},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)

				require.NotNil(t, status.LastPromotion)
				assert.Equal(t, "failed-promotion", status.LastPromotion.Name)
				assert.Empty(t, status.FreightHistory)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
			},
		},
		{
			name: "handles promotion phase transition",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					CurrentPromotion: &kargoapi.PromotionReference{
						Name: "transitioning-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "transitioning-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseRunning,
						Freight: &kargoapi.FreightReference{
							Name: "test-freight",
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.True(t, hasPendingPromotions)

				assert.Equal(t, "transitioning-promotion", status.CurrentPromotion.Name)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				require.NotNil(t, promotingCond)
				assert.Equal(t, metav1.ConditionTrue, promotingCond.Status)
				assert.Equal(t, "ActivePromotion", promotingCond.Reason)
				assert.Contains(t, promotingCond.Message, "Running")
			},
		},
		{
			name: "highest priority promotion has already been processed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "terminal-promotion",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "terminal-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "test-stage",
					},
					Status: kargoapi.PromotionStatus{
						Phase:      kargoapi.PromotionPhaseSucceeded,
						FinishedAt: &metav1.Time{Time: time.Now()},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, hasPendingPromotions bool, err error) {
				require.NoError(t, err)
				assert.False(t, hasPendingPromotions)
				assert.Nil(t, status.CurrentPromotion)

				promotingCond := conditions.Get(&status, kargoapi.ConditionTypePromoting)
				assert.Nil(t, promotingCond)
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
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Promotion{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &RegularStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(fakeevent.NewEventRecorder(10)),
			}

			status, requeue, err := r.syncPromotions(t.Context(), tt.stage)
			tt.assertions(t, status, requeue, err)
		})
	}
}

func TestRegularStageReconciler_syncFreight(t *testing.T) {
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

			r := &RegularStageReconciler{client: c}

			err := r.syncFreight(t.Context(), testStage)
			testCase.assertions(t, c, err)
		})
	}
}

func TestRegularStageReconciler_assessHealth(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name          string
		stage         *kargoapi.Stage
		checkHealthFn func(ctx context.Context, project, stage string, criteria []health.Criteria) kargoapi.Health
		assertions    func(*testing.T, kargoapi.StageStatus)
	}{
		{
			name: "no last promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: nil,
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
			name: "unsuccessful last promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							Phase:        kargoapi.PromotionPhaseAborted,
							HealthChecks: nil,
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnknown, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, "LastPromotionAborted", healthyCond.Reason)
				assert.Equal(t, "Cannot assess health because last Promotion did not succeed", healthyCond.Message)
			},
		},
		{
			name: "no health checks",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							Phase:        kargoapi.PromotionPhaseSucceeded,
							HealthChecks: nil,
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				assert.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateHealthy, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionTrue, healthyCond.Status)
				assert.Equal(t, kargoapi.ConditionTypeHealthy, healthyCond.Reason)
				assert.Contains(t, healthyCond.Message, "Stage is healthy")
			},
		},
		{
			name: "healthy state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							Phase: kargoapi.PromotionPhaseSucceeded,
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(context.Context, string, string, []health.Criteria) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateHealthy, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionTrue, healthyCond.Status)
				assert.Equal(t, string(kargoapi.HealthStateHealthy), healthyCond.Reason)
				assert.Contains(t, healthyCond.Message, "Stage is healthy")
			},
		},
		{
			name: "unhealthy state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							Phase: kargoapi.PromotionPhaseSucceeded,
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(context.Context, string, string, []health.Criteria) kargoapi.Health {
				return kargoapi.Health{
					Status: kargoapi.HealthStateUnhealthy,
					Issues: []string{
						"issue-1", "issue-2",
					},
				}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnhealthy, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionFalse, healthyCond.Status)
				assert.Equal(t, string(kargoapi.HealthStateUnhealthy), healthyCond.Reason)
				assert.Contains(t, healthyCond.Message, "Stage is unhealthy")
				assert.Contains(t, healthyCond.Message, "2 issues in 1 health check")
			},
		},
		{
			name: "not applicable state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{},
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							Phase: kargoapi.PromotionPhaseSucceeded,
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(context.Context, string, string, []health.Criteria) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateNotApplicable}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateNotApplicable, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				assert.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
			},
		},
		{
			name: "unknown state",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							Phase: kargoapi.PromotionPhaseSucceeded,
							HealthChecks: []kargoapi.HealthCheckStep{
								{
									Uses: "test-check",
								},
							},
						},
					},
				},
			},
			checkHealthFn: func(context.Context, string, string, []health.Criteria) kargoapi.Health {
				return kargoapi.Health{Status: kargoapi.HealthStateUnknown}
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus) {
				require.NotNil(t, status.Health)
				assert.Equal(t, kargoapi.HealthStateUnknown, status.Health.Status)

				healthyCond := conditions.Get(&status, kargoapi.ConditionTypeHealthy)
				require.NotNil(t, healthyCond)
				assert.Equal(t, metav1.ConditionUnknown, healthyCond.Status)
				assert.Equal(t, string(kargoapi.HealthStateUnknown), healthyCond.Reason)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			r := &RegularStageReconciler{
				client: c,
				healthChecker: &health.MockAggregatingChecker{
					CheckFn: tt.checkHealthFn,
				},
			}

			status := r.assessHealth(t.Context(), tt.stage)
			tt.assertions(t, status)
		})
	}
}

func TestRegularStageReconciler_verifyStageFreight(t *testing.T) {
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
					CurrentPromotion: &kargoapi.PromotionReference{
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
					LastPromotion: &kargoapi.PromotionReference{
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

			r := &RegularStageReconciler{
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

func TestRegularStageReconciler_markFreightVerifiedForStage(t *testing.T) {
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

			r := &RegularStageReconciler{
				client:        c,
				healthChecker: &health.MockAggregatingChecker{},
			}

			status, err := r.markFreightVerifiedForStage(t.Context(), tt.stage)
			tt.assertions(t, c, status, err)
		})
	}
}

func TestRegularStageReconciler_recordFreightVerificationEvent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := metav1.Now()
	startTime := metav1.NewTime(now.Add(-1 * time.Hour))
	finishTime := metav1.NewTime(now.Add(-30 * time.Minute))

	baseStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stage",
			Namespace: "test-project",
		},
	}

	baseFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: now,
		},
		Alias: "test-alias",
	}

	baseFreightRef := kargoapi.FreightReference{
		Name: "test-freight",
	}

	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		freightRef kargoapi.FreightReference
		vi         *kargoapi.VerificationInfo
		objects    []client.Object
		assertions func(*testing.T, *fakeevent.EventRecorder)
	}{
		{
			name:       "successful verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:      kargoapi.VerificationPhaseSuccessful,
				StartTime:  &startTime,
				FinishTime: &finishTime,
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, corev1.EventTypeNormal, event.EventType)
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationSucceeded), event.Reason)
				assert.Equal(t, "Freight verification succeeded", event.Message)

				assert.Equal(t, baseStage.Name, event.Annotations[kargoapi.AnnotationKeyEventStageName])
				assert.Equal(t, baseFreight.Alias, event.Annotations[kargoapi.AnnotationKeyEventFreightAlias])
				assert.Equal(
					t,
					startTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationStartTime],
				)
				assert.Equal(
					t,
					finishTime.Format(time.RFC3339),
					event.Annotations[kargoapi.AnnotationKeyEventVerificationFinishTime],
				)
			},
		},
		{
			name:       "failed verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseFailed,
				Message: "verification failed due to metrics",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationFailed), event.Reason)
				assert.Equal(t, "verification failed due to metrics", event.Message)
			},
		},
		{
			name:       "verification with analysis run and promotion",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
				AnalysisRun: &kargoapi.AnalysisRunReference{
					Name:      "test-analysis",
					Namespace: "test-project",
				},
			},
			objects: []client.Object{
				baseFreight,
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "test-project",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyPromotion: "test-promotion",
						},
					},
				},
			},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, "test-analysis", event.Annotations[kargoapi.AnnotationKeyEventAnalysisRunName])
				assert.Equal(t, "test-promotion", event.Annotations[kargoapi.AnnotationKeyEventPromotionName])
			},
		},
		{
			name:       "verification with manual actor override",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
				Actor: "manual-user",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, "manual-user", event.Annotations[kargoapi.AnnotationKeyEventActor])
			},
		},
		{
			name:       "freight not found",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
			},
			objects: []client.Object{
				// Freight does not exist
			},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				// No events should be recorded
				assert.Len(t, recorder.Events, 0)
			},
		},
		{
			name:       "analysis run not found",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseSuccessful,
				AnalysisRun: &kargoapi.AnalysisRunReference{
					Name:      "missing-analysis",
					Namespace: "test-project",
				},
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, "missing-analysis", event.Annotations[kargoapi.AnnotationKeyEventAnalysisRunName])
				// Should still record event even though analysis run wasn't found
				assert.NotContains(t, event.Annotations, kargoapi.AnnotationKeyEventPromotionName)
			},
		},
		{
			name:       "errored verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseError,
				Message: "internal error occurred",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationErrored), event.Reason)
				assert.Equal(t, "internal error occurred", event.Message)
			},
		},
		{
			name:       "aborted verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseAborted,
				Message: "verification was canceled",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationAborted), event.Reason)
				assert.Equal(t, "verification was canceled", event.Message)
			},
		},
		{
			name:       "inconclusive verification",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   kargoapi.VerificationPhaseInconclusive,
				Message: "results were inconclusive",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationInconclusive), event.Reason)
				assert.Equal(t, "results were inconclusive", event.Message)
			},
		},
		{
			name:       "unknown phase",
			stage:      baseStage,
			freightRef: baseFreightRef,
			vi: &kargoapi.VerificationInfo{
				Phase:   "invalid-phase",
				Message: "custom message",
			},
			objects: []client.Object{baseFreight},
			assertions: func(t *testing.T, recorder *fakeevent.EventRecorder) {
				require.Len(t, recorder.Events, 1)

				event := <-recorder.Events
				assert.Equal(t, string(kargoapi.EventTypeFreightVerificationUnknown), event.Reason)
				assert.Equal(t, "custom message", event.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			recorder := fakeevent.NewEventRecorder(10)

			r := &RegularStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(recorder),
			}

			r.recordFreightVerificationEvent(tt.stage, tt.freightRef, tt.vi)
			tt.assertions(t, recorder)
		})
	}
}

func TestRegularStageReconciler_startVerification(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

	tests := []struct {
		name             string
		stage            *kargoapi.Stage
		freightCol       kargoapi.FreightCollection
		req              *kargoapi.VerificationRequest
		objects          []client.Object
		credsDB          credentials.Database
		rolloutsDisabled bool
		assertions       func(*testing.T, client.Client, *kargoapi.VerificationInfo, error)
	}{
		{
			name: "rollouts integration disabled",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "Rollouts integration is disabled")
				assert.Equal(t, startTime, vi.StartTime.Time)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "finds existing analysis run",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed successfully",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "existing-analysis", vi.AnalysisRun.Name)
				// StartTime is the injected reconciliation time; FinishTime is
				// the injected end time stamped when the result is recorded.
				require.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, startTime.Unix(), vi.StartTime.Unix())
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "finds existing analysis run with stage name exceeding max label length",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "this-is-a-very-long-stage-name-that-exceeds-the-label-1c0a17e1",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyStage: "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated", // nolint:lll
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed successfully",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "existing-analysis", vi.AnalysisRun.Name)
				require.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, startTime.Unix(), vi.StartTime.Unix())
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "creates new analysis run",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
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
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				assert.NotNil(t, vi.AnalysisRun)

				// Verify analysis run was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				// Verify stage label is not shortened since stage name is short
				assert.Equal(t, "test-stage", ar.Labels[kargoapi.LabelKeyStage])

				// Verify no annotation is added since stage name doesn't need shortening
				_, hasAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.False(t, hasAnnotation, "Stage annotation should not be present when stage name doesn't need shortening")
			},
		},
		{
			name: "creates new analysis run with stage name exceeding max label length",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
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
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				assert.NotNil(t, vi.AnalysisRun)

				// Verify analysis run was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				// Verify stage label was truncated correctly
				assert.Equal(
					t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-1c0a17e1",
					ar.Labels[kargoapi.LabelKeyStage],
				)

				// Verify annotation contains the full stage name
				fullStageName, hasAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.True(t, hasAnnotation)
				assert.Equal(
					t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
					fullStageName,
				)
			},
		},
		{
			name: "handles reverification with control plane actor",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "test-promotion",
					},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID: "prev-verification",
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
			req: &kargoapi.VerificationRequest{
				ID:           "prev-verification",
				Actor:        "test-user",
				ControlPlane: true,
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, "test-user", vi.Actor)

				// Verify promotion annotation was added
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))
				assert.Equal(t, "test-promotion", ar.Annotations[kargoapi.AnnotationKeyPromotion])

				// Verify no stage annotation is added since stage name doesn't need shortening
				_, hasStageAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.False(t, hasStageAnnotation)
			},
		},
		{
			name: "handles reverification with control plane actor and long stage name",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Name: "test-promotion",
					},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID: "prev-verification",
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
			req: &kargoapi.VerificationRequest{
				ID:           "prev-verification",
				Actor:        "test-user",
				ControlPlane: true,
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.NotEmpty(t, vi.ID)
				assert.Equal(t, "test-user", vi.Actor)

				// Verify analysis run was created
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				// Verify both promotion and stage annotations are present
				assert.Equal(t, "test-promotion", ar.Annotations[kargoapi.AnnotationKeyPromotion])
				fullStageName, hasStageAnnotation := ar.Annotations[kargoapi.AnnotationKeyStage]
				assert.True(t, hasStageAnnotation)
				assert.Equal(
					t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-length-and-should-be-truncated",
					fullStageName,
				)

				// Verify stage label was truncated correctly
				assert.Equal(t,
					"this-is-a-very-long-stage-name-that-exceeds-the-label-1c0a17e1",
					ar.Labels[kargoapi.LabelKeyStage],
				)
			},
		},
		{
			name: "resolves repoCredentials() in verification arguments",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
							{Name: "test-template"},
						},
						Args: []kargoapi.AnalysisRunArgument{
							{
								Name: "token",
								Value: "${{ repoCredentials(" +
									"'https://github.com/example/repo.git', 'git'" +
									").Password }}",
							},
						},
					},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Args: []rolloutsapi.Argument{{Name: "token"}},
					},
				},
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (*credentials.Credentials, error) {
					return &credentials.Credentials{Password: "s3cr3t"}, nil
				},
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhasePending, vi.Phase)
				require.NotNil(t, vi.AnalysisRun)

				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: vi.AnalysisRun.Namespace,
					Name:      vi.AnalysisRun.Name,
				}, ar))

				require.Len(t, ar.Spec.Args, 1)
				assert.Equal(t, "token", ar.Spec.Args[0].Name)
				require.NotNil(t, ar.Spec.Args[0].Value)
				assert.Equal(t, "s3cr3t", *ar.Spec.Args[0].Value)
			},
		},
		{
			name: "surfaces error when repoCredentials() is unavailable",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{
							{Name: "test-template"},
						},
						Args: []kargoapi.AnalysisRunArgument{
							{
								Name: "token",
								Value: "${{ repoCredentials(" +
									"'https://github.com/example/repo.git', 'git'" +
									").Password }}",
							},
						},
					},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-freight",
						Namespace: "fake-project",
					},
				},
				&rolloutsapi.AnalysisTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-template",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisTemplateSpec{
						Args: []rolloutsapi.Argument{{Name: "token"}},
					},
				},
			},
			// credsDB intentionally left nil.
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error building AnalysisRun")
				assert.Contains(t, vi.Message, "repoCredentials is not available")
			},
		},
		{
			name: "handles analysis run build error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					Verification: &kargoapi.Verification{},
				},
			},
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
			},
			objects: []client.Object{
				// Missing Freight object for owner reference
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error building AnalysisRun")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &rolloutsapi.AnalysisRun{}).
				Build()

			r := &RegularStageReconciler{
				client:        c,
				credentialsDB: tt.credsDB,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled:   !tt.rolloutsDisabled,
					RolloutsControllerInstanceID: "test-instance",
				},
			}

			vi, err := r.startVerification(t.Context(), tt.stage, tt.freightCol, tt.req, startTime, fixedEndTime)
			tt.assertions(t, c, vi, err)
		})
	}
}

func TestRegularStageReconciler_getVerificationResult(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	endTime := now.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

	tests := []struct {
		name             string
		freight          kargoapi.FreightCollection
		objects          []client.Object
		rolloutsDisabled bool
		assertions       func(*testing.T, *kargoapi.VerificationInfo, error)
	}{
		{
			name: "error when no current verification info",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no current verification info")
				assert.Nil(t, vi)
			},
		},
		{
			name: "error when no analysis run reference",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no AnalysisRun reference")
				assert.Nil(t, vi)
			},
		},
		{
			name: "error when rollouts integration disabled",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Contains(t, vi.Message, "Rollouts integration is disabled")
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "error when analysis run not found",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "missing-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.True(t, apierrors.IsNotFound(err))

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Contains(t, vi.Message, "error getting AnalysisRun")
				assert.NotNil(t, vi.AnalysisRun)
				assert.Equal(t, "missing-analysis", vi.AnalysisRun.Name)
			},
		},
		{
			name: "preserves actor in verification info",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						Actor:     "test-user",
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Running",
						Message: "Analysis in progress",
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "test-user", vi.Actor)
				assert.Equal(t, kargoapi.VerificationPhaseRunning, vi.Phase)
			},
		},
		{
			name: "handles successful analysis run",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:       rolloutsapi.AnalysisPhaseSuccessful,
						Message:     "Analysis completed successfully",
						CompletedAt: &metav1.Time{Time: endTime},
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: endTime},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "Analysis completed successfully", vi.Message)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "handles failed analysis run",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:       "Failed",
						Message:     "Something went wrong",
						CompletedAt: &metav1.Time{Time: endTime},
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: endTime},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "Something went wrong", vi.Message)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
				assert.Equal(t, string(rolloutsapi.AnalysisPhaseFailed), vi.AnalysisRun.Phase)
			},
		},
		{
			name: "handles error analysis run",
			freight: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:       rolloutsapi.AnalysisPhaseError,
						Message:     "Something went wrong",
						CompletedAt: &metav1.Time{Time: endTime},
						MetricResults: []rolloutsapi.MetricResult{
							{
								Measurements: []rolloutsapi.Measurement{
									{
										FinishedAt: &metav1.Time{Time: endTime},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
				assert.Equal(t, "Something went wrong", vi.Message)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
				assert.Equal(t, string(rolloutsapi.AnalysisPhaseError), vi.AnalysisRun.Phase)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &rolloutsapi.AnalysisRun{}).
				Build()

			r := &RegularStageReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				},
				backoffCfg: wait.Backoff{
					Duration: 1 * time.Second,
					Factor:   2,
					Steps:    2,
					Cap:      1 * time.Second,
					Jitter:   0.1,
				},
			}

			vi, err := r.getVerificationResult(t.Context(), tt.freight, fixedEndTime)
			tt.assertions(t, vi, err)
		})
	}
}

func TestRegularStageReconciler_abortVerification(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	endTime := now.Add(5 * time.Minute)
	fixedEndTime := func() time.Time { return endTime }

	tests := []struct {
		name             string
		freightCol       kargoapi.FreightCollection
		req              *kargoapi.VerificationRequest
		objects          []client.Object
		rolloutsDisabled bool
		interceptor      interceptor.Funcs
		assertions       func(*testing.T, client.Client, *kargoapi.VerificationInfo, error)
	}{
		{
			name: "error when no current verification info",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no current verification info")
				assert.Nil(t, vi)
			},
		},
		{
			name: "error when no analysis run reference",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.ErrorContains(t, err, "no AnalysisRun reference")
				assert.Nil(t, vi)
			},
		},
		{
			name: "returns current verification if already terminal",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseSuccessful,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseSuccessful, vi.Phase)
				assert.Equal(t, "test-verification", vi.ID)
			},
		},
		{
			name: "error when rollouts integration disabled",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			rolloutsDisabled: true,
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "Rollouts integration is disabled")
				assert.Equal(t, "test-verification", vi.ID)
				assert.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
			},
		},
		{
			name: "handles patch error",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
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
					return fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err) // Error is captured in verification info
				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error terminating AnalysisRun")
			},
		},
		{
			name: "successfully aborts verification",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisRunSpec{
						Metrics: []rolloutsapi.Metric{
							{Name: "test-metric"},
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Running",
						Message: "Analysis in progress",
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
				assert.Equal(t, "Verification aborted by user", vi.Message)
				assert.Equal(t, "test-verification", vi.ID)
				assert.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
				assert.Equal(t, "test-analysis", vi.AnalysisRun.Name)

				// Verify analysis run was patched with terminate = true
				ar := &rolloutsapi.AnalysisRun{}
				require.NoError(t, c.Get(t.Context(), types.NamespacedName{
					Namespace: "fake-project",
					Name:      "test-analysis",
				}, ar))
				assert.True(t, ar.Spec.Terminate)
			},
		},
		{
			name: "handles already terminated analysis run",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
					Spec: rolloutsapi.AnalysisRunSpec{
						Terminate: true,
						Metrics: []rolloutsapi.Metric{
							{Name: "test-metric"},
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase:   "Successful",
						Message: "Analysis completed",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
				assert.Equal(t, "Verification aborted by user", vi.Message)
				assert.Equal(t, "test-verification", vi.ID)
				assert.NotNil(t, vi.StartTime)
				require.NotNil(t, vi.FinishTime)
				assert.Equal(t, endTime.Unix(), vi.FinishTime.Unix())
				assert.Equal(t, "test-analysis", vi.AnalysisRun.Name)
			},
		},
		{
			name: "sets actor in verification info",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "test-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			req: &kargoapi.VerificationRequest{
				ID:    "test-verification",
				Actor: "test-user",
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, "test-user", vi.Actor)
				assert.Equal(t, kargoapi.VerificationPhaseFailed, vi.Phase)
			},
		},
		{
			name: "handles analysis run not found",
			freightCol: kargoapi.FreightCollection{
				ID: "test-collection",
				Freight: map[string]kargoapi.FreightReference{
					"warehouse": {Name: "test-freight"},
				},
				VerificationHistory: []kargoapi.VerificationInfo{
					{
						ID:        "test-verification",
						Phase:     kargoapi.VerificationPhaseRunning,
						StartTime: &metav1.Time{Time: now},
						AnalysisRun: &kargoapi.AnalysisRunReference{
							Name:      "missing-analysis",
							Namespace: "fake-project",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ client.Client, vi *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)

				require.NotNil(t, vi)
				assert.Equal(t, kargoapi.VerificationPhaseError, vi.Phase)
				assert.Contains(t, vi.Message, "error terminating AnalysisRun")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}, &rolloutsapi.AnalysisRun{})

			if tt.interceptor.Patch != nil {
				builder = builder.WithInterceptorFuncs(tt.interceptor)
			}

			c := builder.Build()

			r := &RegularStageReconciler{
				client: c,
				cfg: ReconcilerConfig{
					RolloutsIntegrationEnabled: !tt.rolloutsDisabled,
				},
			}

			vi, err := r.abortVerification(t.Context(), tt.freightCol, tt.req, fixedEndTime)
			tt.assertions(t, c, vi, err)
		})
	}
}

func TestRegularStageReconciler_findExistingAnalysisRun(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	tests := []struct {
		name         string
		stage        types.NamespacedName
		freightColID string
		objects      []client.Object
		interceptor  interceptor.Funcs
		assertions   func(*testing.T, *rolloutsapi.AnalysisRun, error)
	}{
		{
			name: "no analysis runs found",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)
				assert.Nil(t, ar)
			},
		},
		{
			name: "handles list error",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			interceptor: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return fmt.Errorf("list error")
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.ErrorContains(t, err, "error listing AnalysisRuns")
				assert.Nil(t, ar)
			},
		},
		{
			name: "finds most recent analysis run",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "older-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase: "Successful",
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "newer-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
					Status: rolloutsapi.AnalysisRunStatus{
						Phase: "Failed",
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "newer-analysis", ar.Name)
				assert.Equal(t, hourAgo.Unix(), ar.CreationTimestamp.Unix())
			},
		},
		{
			name: "filters by correct stage",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "other-stage-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "other-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-stage-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "correct-stage-analysis", ar.Name)
			},
		},
		{
			name: "filters by correct freight collection",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "other-freight-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "other-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-freight-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "correct-freight-analysis", ar.Name)
			},
		},
		{
			name: "handles multiple namespaces correctly",
			stage: types.NamespacedName{
				Namespace: "test-namespace",
				Name:      "test-stage",
			},
			freightColID: "test-collection",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "wrong-namespace-analysis",
						Namespace:         "fake-project",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "correct-namespace-analysis",
						Namespace:         "test-namespace",
						CreationTimestamp: metav1.Time{Time: twoHoursAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "test-collection",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "test-namespace", ar.Namespace)
				assert.Equal(t, "correct-namespace-analysis", ar.Name)
			},
		},
		{
			name: "empty freight collection ID",
			stage: types.NamespacedName{
				Namespace: "fake-project",
				Name:      "test-stage",
			},
			freightColID: "",
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage:             "test-stage",
							kargoapi.LabelKeyFreightCollection: "",
						},
					},
				},
			},
			assertions: func(t *testing.T, ar *rolloutsapi.AnalysisRun, err error) {
				require.NoError(t, err)

				require.NotNil(t, ar)
				assert.Equal(t, "test-analysis", ar.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithStatusSubresource(&rolloutsapi.AnalysisRun{})

			if tt.interceptor.List != nil {
				builder = builder.WithInterceptorFuncs(tt.interceptor)
			}

			c := builder.Build()

			r := &RegularStageReconciler{client: c}

			ar, err := r.findExistingAnalysisRun(t.Context(), tt.stage, tt.freightColID)
			tt.assertions(t, ar, err)
		})
	}
}

func TestRegularStageReconciler_autoPromoteFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *fakeevent.EventRecorder, client.Client, kargoapi.StageStatus, error)
	}{
		{
			name: "no requested freight",
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "auto-promotion not allowed",
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
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: false,
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "disabling auto-promotion clears active holds",
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
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/test-warehouse": {
							FreightName: "old-freight",
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: false,
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
				// The hold that was present in Stage status must be cleared.
				assert.Empty(t, status.AutoPromotionHolds)

				// Verify no promotions were created
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "projectconfig not found",
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "handles direct freight from warehouse",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "sorts by discoveredAt when set",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "falls back to creationTimestamp when discoveredAt is unset",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "skips promotion when current freight is latest",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Empty(t, promoList.Items)
			},
		},
		{
			name: "skips promotion if a non-terminal one already exists",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			// A fast Promotion can go from pending to succeeded in the interval
			// between syncPromotions observing it and autoPromoteFreight acting.
			// Its outcome is not recorded yet (it is newer than
			// status.lastPromotion), so auto-promotion must stand down rather
			// than create a duplicate.
			name: "skips promotion when a succeeded one for the candidate is not yet recorded",
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
				// No LastPromotion: the succeeded Promotion below has not been
				// processed by syncPromotions.
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			// The counterpart of the case above: a RECORDED succeeded Promotion
			// for the candidate (not newer than status.lastPromotion) must not
			// block. With no hold in play, auto-promotion owns the origin; the
			// old "any Promotion already exists" guard stays removed.
			name: "proceeds when a succeeded promotion for the candidate has been recorded",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					// Lexically greater than the Promotion's name below: its
					// outcome has been recorded. The Stage does not currently
					// have the candidate (e.g. it was later moved elsewhere and
					// any hold has since been evicted).
					LastPromotion: &kargoapi.PromotionReference{Name: "zz-last-promotion"},
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "recorded-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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

				// A new promotion for the candidate was created.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 2)
				var created int
				for _, promo := range promoList.Items {
					if promo.Name != "recorded-promotion" {
						created++
						assert.Equal(t, "test-freight-1", promo.Spec.Freight)
					}
				}
				assert.Equal(t, 1, created)
			},
		},
		{
			name: "skips promotion if the last terminal one was not successful",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "existing-promotion",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseErrored,
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
				assert.Equal(t, "existing-promotion", promoList.Items[0].Name)
			},
		},
		{
			name: "skips when newest terminal promotion failed even if an older one succeeded",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{Direct: true},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
			},
			objects: terminalPromotionOrderingObjects(
				kargoapi.PromotionPhaseSucceeded,
				kargoapi.PromotionPhaseErrored,
			),
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				assert.True(t, status.AutoPromotionEnabled)

				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 2)
			},
		},
		{
			name: "allows MatchUpstream when newest terminal promotion succeeded even if an older one failed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Status: kargoapi.StageStatus{
					// Both terminal Promotions below have had their outcomes
					// recorded: neither is newer than the last one processed.
					LastPromotion: &kargoapi.PromotionReference{Name: "zz-last-promotion"},
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{
							Stages: []string{"upstream-stage"},
							AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
								SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							},
						},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
			},
			objects: terminalPromotionOrderingObjects(
				kargoapi.PromotionPhaseErrored,
				kargoapi.PromotionPhaseSucceeded,
			),
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				assert.True(t, status.AutoPromotionEnabled)

				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 3)
				for _, promo := range promoList.Items {
					if promo.Name == "older-promotion" || promo.Name == "newer-promotion" {
						continue
					}
					assert.Equal(t, "test-freight-1", promo.Spec.Freight)
				}
			},
		},
		{
			name: "skips MatchUpstream when newest terminal promotion failed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{
							Stages: []string{"upstream-stage"},
							AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
								SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							},
						},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
			},
			objects: terminalPromotionOrderingObjects(
				kargoapi.PromotionPhaseSucceeded,
				kargoapi.PromotionPhaseErrored,
			),
			assertions: func(
				t *testing.T,
				_ *fakeevent.EventRecorder,
				c client.Client,
				status kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				assert.True(t, status.AutoPromotionEnabled)

				// Newest terminal Promotion errored; failure-loop prevention must
				// apply to MatchUpstream just as it does to NewestFreight.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 2,
					"no new Promotion should be created when newest terminal failed")
			},
		},
		{
			name: "handles verified freight from upstream stages",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			// Regression: a hold must prevent auto-promotion even when the most
			// recent terminal Promotion for the same Freight would allow retry.
			name: "active hold blocks auto-promotion",
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
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/test-warehouse": {
							FreightName: "older-freight",
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "test-warehouse",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
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
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "hold-succeeded-promotion",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels:            map[string]string{kargoapi.LabelKeyStage: "test-stage"},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-1",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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

				// The creation gate must leave the pre-existing terminal Promotion
				// as the only one: no new (doomed) Promotion was created.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, kargoapi.PromotionPhaseSucceeded, promoList.Items[0].Status.Phase)
			},
		},
		{
			name: "handles verified freight from upstream stages with soak time requirement",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-2", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "handles freight approved for stage",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "test-freight-1", promoList.Items[0].Spec.Freight)
			},
		},
		{
			name: "handles multiple freight requests",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
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
			name: "creates promotion with events",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
			assertions: func(
				t *testing.T,
				e *fakeevent.EventRecorder,
				c client.Client,
				_ kargoapi.StageStatus,
				err error,
			) {
				require.NoError(t, err)
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				require.Len(t, promoList.Items, 1)
				require.Len(t, e.Events, 1)

				event := <-e.Events
				assert.Equal(t, corev1.EventTypeNormal, event.EventType)
				assert.Equal(t, "PromotionCreated", event.Reason)
				assert.Contains(t, event.Message, "Automatically promoted Freight")
				assert.NotEmpty(t, event.Annotations)
			},
		},
		{
			name: "deduplicates freight from multiple sources",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				assert.Len(t, promoList.Items, 1)
			},
		},
		{
			name: "handles promotion creation error",
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
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								Stage:                "test-stage",
								AutoPromotionEnabled: true,
							},
						},
					},
				},
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
				require.ErrorContains(t, err, "error creating Promotion")
			},
		},
		{
			name: "skips promotion when origin has an AutoPromotionHold",
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
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/test-warehouse": {
							FreightName: "test-freight-old",
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-new",
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

				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(
					t.Context(), promoList, client.InNamespace("fake-project"),
				))
				assert.Empty(
					t, promoList.Items,
					"hold on origin should suppress auto-promotion",
				)
			},
		},
		{
			name: "skips promotion when origin has a pending hold-intent Promotion",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{Direct: true},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "hold-promo",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/test-warehouse",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-old",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
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

				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(
					t.Context(), promoList, client.InNamespace("fake-project"),
				))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "hold-promo", promoList.Items[0].Name)
			},
		},
		{
			// A hold-intent Promotion that reached a terminal phase other than
			// Succeeded never establishes a hold, so it must not block
			// auto-promotion even before it is recorded -- an unsuccessful
			// Promotion may go unrecorded for a long time (e.g. one aborted
			// before ever being acknowledged).
			name: "proceeds when an unrecorded hold-intent Promotion failed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{Direct: true},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				// No hold in status and no LastPromotion: the failed Promotion
				// below has not been processed by syncPromotions.
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "failed-hold-promo",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/test-warehouse",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-old",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseFailed,
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

				// A new auto-promotion for the candidate was created; the
				// failed hold-intent Promotion did not block it.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(
					t.Context(), promoList, client.InNamespace("fake-project"),
				))
				require.Len(t, promoList.Items, 2)
				var created int
				for _, promo := range promoList.Items {
					if promo.Name != "failed-hold-promo" {
						created++
						assert.Equal(t, "test-freight-new", promo.Spec.Freight)
					}
				}
				assert.Equal(t, 1, created)
			},
		},
		{
			// A Promotion can succeed in the interval between syncPromotions
			// computing hold state and autoPromoteFreight acting on it. Its
			// hold is not recorded yet (it is newer than status.lastPromotion),
			// but auto-promotion must already stand down for the origin.
			name: "skips promotion when a succeeded hold-intent Promotion is not yet recorded",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{Direct: true},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				// No hold in status and no LastPromotion: the succeeded
				// Promotion below has not been processed by syncPromotions.
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "hold-promo",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/test-warehouse",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-old",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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

				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(
					t.Context(), promoList, client.InNamespace("fake-project"),
				))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "hold-promo", promoList.Items[0].Name)
			},
		},
		{
			// The counterpart of the case above: once syncPromotions has
			// recorded a hold-intent Promotion's outcome (it is not newer than
			// status.lastPromotion) and no hold is active (e.g. it was later
			// released), the mere existence of that old Promotion must not
			// block auto-promotion.
			name: "proceeds when a recorded hold-intent Promotion exists but no hold is active",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{Direct: true},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				Status: kargoapi.StageStatus{
					// LastPromotion is lexically greater than the hold-intent
					// Promotion's name: its outcome has been recorded, and no
					// hold remains in status.
					LastPromotion: &kargoapi.PromotionReference{
						Name: "test-stage.2-release-promo",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "test-freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-stage.1-hold-promo",
						Annotations: map[string]string{
							kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/test-warehouse",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "test-freight-old",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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

				// A new auto-promotion for the candidate was created.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(
					t.Context(), promoList, client.InNamespace("fake-project"),
				))
				require.Len(t, promoList.Items, 2)
				var created int
				for _, promo := range promoList.Items {
					if promo.Name != "test-stage.1-hold-promo" {
						created++
						assert.Equal(t, "test-freight-new", promo.Spec.Freight)
					}
				}
				assert.Equal(t, 1, created)
			},
		},
		{
			name: "continues promotion for unheld origin on multi-origin stage",
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
								Name: "held-warehouse",
							},
							Sources: kargoapi.FreightSources{Direct: true},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "open-warehouse",
							},
							Sources: kargoapi.FreightSources{Direct: true},
						},
					},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				Status: kargoapi.StageStatus{
					AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
						"Warehouse/held-warehouse": {
							FreightName: "held-freight-old",
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "held-warehouse",
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "open-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "held-freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "held-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "open-freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "open-warehouse",
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

				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(
					t.Context(), promoList, client.InNamespace("fake-project"),
				))
				require.Len(t, promoList.Items, 1)
				assert.Equal(t, "open-freight-new", promoList.Items[0].Spec.Freight)
			},
		},
		{
			// After a user clears a hold by promoting the candidate, the Stage
			// has no active hold and a newer Freight has since arrived.
			// Auto-promotion must fire for the new candidate.
			name: "auto-promotes new freight after hold is cleared (NewestFreight)",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{Direct: true},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				Status: kargoapi.StageStatus{
					// freight-old is currently deployed; no active hold.
					FreightHistory: kargoapi.FreightHistory{{
						Freight: map[string]kargoapi.FreightReference{
							"Warehouse/test-warehouse": {Name: "freight-old"},
						},
					}},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-old",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				// The hold-clearing Promotion that put the Stage at freight-old.
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "hold-clearing-promotion",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-old",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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

				// Auto-promotion must have fired for the new candidate.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				newPromos := make([]kargoapi.Promotion, 0)
				for _, p := range promoList.Items {
					if p.Name != "hold-clearing-promotion" {
						newPromos = append(newPromos, p)
					}
				}
				require.Len(t, newPromos, 1, "auto-promotion should fire after hold is cleared")
				assert.Equal(t, "freight-new", newPromos[0].Spec.Freight)
			},
		},
		{
			// Same as the NewestFreight case but with MatchUpstream policy. The
			// upstream Stage has advanced to a newer Freight while the downstream
			// held; after the hold is cleared the downstream should follow.
			name: "auto-promotes new upstream freight after hold is cleared (MatchUpstream)",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{{
						Origin: kargoapi.FreightOrigin{
							Kind: kargoapi.FreightOriginKindWarehouse,
							Name: "test-warehouse",
						},
						Sources: kargoapi.FreightSources{
							Stages: []string{"upstream-stage"},
							AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
								SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
							},
						},
					}},
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fake-step"}},
						},
					},
				},
				Status: kargoapi.StageStatus{
					// freight-old is currently deployed; no active hold.
					FreightHistory: kargoapi.FreightHistory{{
						Freight: map[string]kargoapi.FreightReference{
							"Warehouse/test-warehouse": {Name: "freight-old"},
						},
					}},
				},
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "test-stage",
							AutoPromotionEnabled: true,
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
						Name:      "test-warehouse",
					},
				},
				// freight-old is no longer in the upstream Stage.
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-old",
						CreationTimestamp: metav1.Time{Time: hourAgo},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
				},
				// freight-new is the current upstream candidate.
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "freight-new",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "test-warehouse",
					},
					Status: kargoapi.FreightStatus{
						CurrentlyIn: map[string]kargoapi.CurrentStage{
							"upstream-stage": {},
						},
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
				// The hold-clearing Promotion that put the Stage at freight-old.
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:         "fake-project",
						Name:              "hold-clearing-promotion",
						CreationTimestamp: metav1.Time{Time: hourAgo},
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "test-stage",
						Freight: "freight-old",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
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

				// Auto-promotion must follow the upstream after the hold clears.
				promoList := &kargoapi.PromotionList{}
				require.NoError(t, c.List(t.Context(), promoList, client.InNamespace("fake-project")))
				newPromos := make([]kargoapi.Promotion, 0)
				for _, p := range promoList.Items {
					if p.Name != "hold-clearing-promotion" {
						newPromos = append(newPromos, p)
					}
				}
				require.Len(t, newPromos, 1, "auto-promotion should follow upstream after hold is cleared")
				assert.Equal(t, "freight-new", newPromos[0].Spec.Freight)
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
					&kargoapi.Promotion{},
					indexer.PromotionsByStageField,
					indexer.PromotionsByStage,
				).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.PromotionsByStageAndFreightField,
					indexer.PromotionsByStageAndFreight,
				).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.PromotionsByTerminalField,
					indexer.PromotionsByTerminal,
				).
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
				)

			c := builder.Build()
			recorder := fakeevent.NewEventRecorder(5)

			r := &RegularStageReconciler{
				client:      c,
				eventSender: k8sevent.NewEventSender(recorder),
			}

			status, err := r.autoPromoteFreight(t.Context(), tt.stage)
			tt.assertions(t, recorder, c, status, err)
		})
	}
}

func terminalPromotionOrderingObjects(
	olderPhase kargoapi.PromotionPhase,
	newerPhase kargoapi.PromotionPhase,
) []client.Object {
	newerTime := time.Now()
	olderTime := newerTime.Add(-time.Hour)
	return []client.Object{
		&kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fake-project",
				Namespace: "fake-project",
			},
			Spec: kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{{
					Stage:                "test-stage",
					AutoPromotionEnabled: true,
				}},
			},
		},
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
				CreationTimestamp: metav1.Time{Time: newerTime},
			},
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKindWarehouse,
				Name: "test-warehouse",
			},
			Status: kargoapi.FreightStatus{
				CurrentlyIn: map[string]kargoapi.CurrentStage{
					"upstream-stage": {},
				},
				VerifiedIn: map[string]kargoapi.VerifiedStage{
					"upstream-stage": {},
				},
			},
		},
		&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "fake-project",
				Name:              "older-promotion",
				CreationTimestamp: metav1.Time{Time: olderTime},
				Labels: map[string]string{
					kargoapi.LabelKeyStage: "test-stage",
				},
			},
			Spec: kargoapi.PromotionSpec{
				Stage:   "test-stage",
				Freight: "test-freight-1",
			},
			Status: kargoapi.PromotionStatus{Phase: olderPhase},
		},
		&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "fake-project",
				Name:              "newer-promotion",
				CreationTimestamp: metav1.Time{Time: newerTime},
				Labels: map[string]string{
					kargoapi.LabelKeyStage: "test-stage",
				},
			},
			Spec: kargoapi.PromotionSpec{
				Stage:   "test-stage",
				Freight: "test-freight-1",
			},
			Status: kargoapi.PromotionStatus{Phase: newerPhase},
		},
	}
}

func Test_summarizeConditions(t *testing.T) {
	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		status     *kargoapi.StageStatus
		err        error
		assertions func(*testing.T, *kargoapi.StageStatus)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "Promoting", readyCond.Reason)
				assert.Equal(t, "Stage is promoting", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
				LastPromotion: &kargoapi.PromotionReference{
					Status: &kargoapi.PromotionStatus{
						Phase:   kargoapi.PromotionPhaseFailed,
						Message: "Promotion failed due to error",
					},
				},
			},
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "LastPromotionFailed", readyCond.Reason)
				assert.Equal(t, "Promotion failed due to error", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)

				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "HealthCheckFailed", readyCond.Reason)
				assert.Equal(t, "Health check failed", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "Unhealthy", readyCond.Reason)
				assert.Equal(t, "Stage is not healthy", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "HealthCheckPending", readyCond.Reason)
				assert.Equal(t, "Health check in progress", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "VerificationPending", readyCond.Reason)
				assert.Equal(t, "Verification is pending", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "VerificationError", readyCond.Reason)
				assert.Equal(t, "Verification failed", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
				assert.Equal(t, "PendingVerification", readyCond.Reason)
				assert.Equal(t, "Stage is not verified", readyCond.Message)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)

				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
				assert.Equal(t, "Verified", readyCond.Reason)
				assert.Equal(t, "Stage is verified", readyCond.Message)
				assert.Equal(t, int64(1), readyCond.ObservedGeneration)

				assert.Equal(t, int64(1), status.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				readyCond := conditions.Get(status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)

				reconcileCond := conditions.Get(status, kargoapi.ConditionTypeReconciling)
				assert.Nil(t, reconcileCond, "Reconciling condition should be deleted when ready")

				assert.Equal(t, int64(1), status.ObservedGeneration)
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
			assertions: func(t *testing.T, status *kargoapi.StageStatus) {
				assert.Equal(t, "1/2 Fulfilled", status.FreightSummary)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summarizeConditions(tt.stage, tt.status, tt.err)
			tt.assertions(t, tt.status)
		})
	}
}

func Test_buildFreightSummary(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		current   *kargoapi.FreightCollection
		expected  string
	}{
		{
			name:      "nil current",
			requested: 2,
			current:   nil,
			expected:  "0/2 Fulfilled",
		},
		{
			name:      "single freight",
			requested: 1,
			current: &kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"test": {Name: "test-freight"},
				},
			},
			expected: "test-freight",
		},
		{
			name:      "multiple freight",
			requested: 3,
			current: &kargoapi.FreightCollection{
				Freight: map[string]kargoapi.FreightReference{
					"test1": {Name: "test-freight-1"},
					"test2": {Name: "test-freight-2"},
				},
			},
			expected: "2/3 Fulfilled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFreightSummary(tt.requested, tt.current)
			assert.Equal(t, tt.expected, result)
		})
	}
}
