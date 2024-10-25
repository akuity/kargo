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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	fakeevent "github.com/akuity/kargo/internal/kubernetes/event/fake"
)

func Test_controlFlowStageReconciler_Reconcile(t *testing.T) {
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
			name: "ignores non-control flow stage",
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
					// Not a control flow stage
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{},
							},
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
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "default",
					Name:             "test-stage",
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
					Name:             "test-stage",
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
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.True(t, result.Requeue)

				// Verify finalizer was added
				stage := &kargoapi.Stage{}
				err = c.Get(context.Background(), types.NamespacedName{
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
					Name:      "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
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
				err = c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				}, stage)
				require.NoError(t, err)
				assert.Contains(t, stage.Status.Message, "something went wrong")
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
					Name:      "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
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
			name: "status update error after successful reconcile",
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				},
			},
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "default",
					Name:      "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
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
				require.ErrorContains(t, err, "failed to update Stage status: status update error")
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
					Name:      "test-stage",
					Finalizers: []string{kargoapi.FinalizerName},
				},
				Spec: kargoapi.StageSpec{
				},
			},
			assertions: func(t *testing.T, c client.Client, result ctrl.Result, err error) {
				require.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)

				// Verify status was updated
				stage := &kargoapi.Stage{}
				err = c.Get(context.Background(), types.NamespacedName{
					Namespace: "default",
					Name:      "test-stage",
				}, stage)
				require.NoError(t, err)
				assert.Equal(t, kargoapi.StagePhaseNotApplicable, stage.Status.Phase)
				assert.Empty(t, stage.Status.Message)
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
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByWarehouseIndexField,
					indexer.FreightByWarehouseIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesIndexField,
					indexer.FreightByVerifiedStagesIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesIndexField,
					indexer.FreightApprovedForStagesIndexer,
				).
				WithStatusSubresource(&kargoapi.Stage{}, &kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &controlFlowStageReconciler{
				client:        c,
				eventRecorder: fakeevent.NewEventRecorder(10),
			}

			result, err := r.Reconcile(context.Background(), tt.req)
			tt.assertions(t, c, result, err)
		})
	}
}

func Test_controlFlowStageReconciler_reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	startTime := time.Now()

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, kargoapi.StageStatus, error)
	}{
		{
			name: "successful reconciliation with no freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.StagePhaseNotApplicable, status.Phase)
				assert.Empty(t, status.Message)
			},
		},
		{
			name: "successful reconciliation with new freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.StagePhaseNotApplicable, status.Phase)
				assert.Empty(t, status.Message)
			},
		},
		{
			name: "error getting available freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Contains(t, status.Message, "something went wrong")
			},
		},
		{
			name: "error verifying freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
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
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.ErrorContains(t, err, "failed to verify 1 Freight")
				assert.Contains(t, status.Message, "failed to verify 1 Freight")
			},
		},
		{
			name: "already verified freight",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.StagePhaseNotApplicable, status.Phase)
				assert.Empty(t, status.Message)
			},
		},
		{
			name: "handles stage with refresh annotation",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "default",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: "refresh-token",
					},
				},
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "warehouse-1",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.StagePhaseNotApplicable, status.Phase)
				assert.Equal(t, "refresh-token", status.LastHandledRefresh)
				assert.Empty(t, status.Message)
			},
		},
		{
			name: "observes generation on reconciliation",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace: "default",
					Generation: 2,
				},
				Status: kargoapi.StageStatus{
					ObservedGeneration: 1,
				},
			},
			assertions: func(t *testing.T, status kargoapi.StageStatus, err error) {
				require.NoError(t, err)
				assert.Equal(t, int64(2), status.ObservedGeneration)
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
					indexer.FreightByWarehouseIndexField,
					indexer.FreightByWarehouseIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesIndexField,
					indexer.FreightByVerifiedStagesIndexer,
				).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &controlFlowStageReconciler{
				client: c,
				eventRecorder: fakeevent.NewEventRecorder(10),
			}

			status, err := r.reconcile(context.Background(), tt.stage, startTime)
			tt.assertions(t, status, err)
		})
	}
}

func Test_controlFlowStageReconciler_initializeStatus(t *testing.T) {
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
				assert.Equal(t, kargoapi.StagePhaseNotApplicable, newStatus.Phase)
				assert.Equal(t, int64(2), newStatus.ObservedGeneration)
			},
		},
		{
			name: "resets previous message",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					Message: "previous message",
				},
			},
			assertions: func(t *testing.T, newStatus kargoapi.StageStatus) {
				assert.Empty(t, newStatus.Message)
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
			newStatus := (&controlFlowStageReconciler{}).initializeStatus(tt.stage)
			tt.assertions(t, newStatus)
		})
	}
}

func Test_controlFlowStageReconciler_getAvailableFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      types.NamespacedName
		objects    []client.Object
		requested  []kargoapi.FreightRequest
		interceptor interceptor.Funcs
		assertions func(*testing.T, []kargoapi.Freight, error)
	}{
		{
			name: "no freight requests returns empty list",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			requested: []kargoapi.FreightRequest{},
			assertions: func(t *testing.T, got []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "direct warehouse freight",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "other-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-2",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-2",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
				},
			},
			requested: []kargoapi.FreightRequest{
				{
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
				},
			},
			assertions: func(t *testing.T, got []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Len(t, got, 2)
				assert.Equal(t, "freight-1", got[0].Name)
				assert.Equal(t, "freight-2", got[1].Name)
			},
		},
		{
			name: "ignores already verified direct warehouse freight",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"test-stage": {},
						},
					},
				},
			},
			requested: []kargoapi.FreightRequest{
				{
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
				},
			},
			assertions: func(t *testing.T, got []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		{
			name: "upstream warehouse freight",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "other-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"other-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-2",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
			},
			requested: []kargoapi.FreightRequest{
				{
					Sources: kargoapi.FreightSources{
						Stages: []string{"upstream-stage"},
					},
				},
			},
			assertions: func(t *testing.T, freights []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Len(t, freights, 2)
				assert.Equal(t, "freight-1", freights[0].Name)
				assert.Equal(t, "freight-2", freights[1].Name)
			},
		},
		{
			name: "ignores already verified upstream warehouse freight",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
							"test-stage":     {},
						},
					},
				},
			},
			requested: []kargoapi.FreightRequest{
				{
					Sources: kargoapi.FreightSources{
						Stages: []string{"upstream-stage"},
					},
				},
			},
			assertions: func(t *testing.T, freights []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Empty(t, freights)
			},
		},
		{
			name: "multiple freight requests",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "direct-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "upstream-freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-2",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "upstream-freight-2",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-2",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
			},
			requested: []kargoapi.FreightRequest{
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
					Sources: kargoapi.FreightSources{
						Stages: []string{"upstream-stage"},
					},
				},
			},
			assertions: func(t *testing.T, freights []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Len(t, freights, 3)
				assert.Equal(t, "direct-freight-1", freights[0].Name)
				assert.Equal(t, "upstream-freight-1", freights[1].Name)
				assert.Equal(t, "upstream-freight-2", freights[2].Name)
			},
		},
		{
			name: "deduplicates freight",
			stage: types.NamespacedName{
				Namespace: "default",
			},
			objects: []client.Object{
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "freight-1",
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Status: kargoapi.FreightStatus{
						VerifiedIn: map[string]kargoapi.VerifiedStage{
							"upstream-stage": {},
						},
					},
				},
			},
			requested: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
						Stages: []string{"upstream-stage"},
					},
				},
			},
			assertions: func(t *testing.T, freights []kargoapi.Freight, err error) {
				require.NoError(t, err)
				assert.Len(t, freights, 1)
				assert.Equal(t, "freight-1", freights[0].Name)
			},
		},
		{
			name: "warehouse list error",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			requested: []kargoapi.FreightRequest{
				{
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: "warehouse-1",
					},
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
					case strings.Contains(listOpts.FieldSelector.String(), indexer.FreightByWarehouseIndexField):
						return fmt.Errorf("something went wrong")
					default:
						return c.List(ctx, list, opts...)
					}
				},
			},
			assertions: func(t *testing.T, got []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Nil(t, got)
			},
		},
		{
			name: "stage list error",
			stage: types.NamespacedName{
				Namespace: "default",
				Name:      "test-stage",
			},
			requested: []kargoapi.FreightRequest{
				{
					Sources: kargoapi.FreightSources{
						Stages: []string{"upstream-stage"},
					},
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
					case strings.Contains(listOpts.FieldSelector.String(), indexer.FreightByVerifiedStagesIndexField):
						return fmt.Errorf("something went wrong")
					default:
						return c.List(ctx, list, opts...)
					}
				},
			},
			assertions: func(t *testing.T, got []kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Nil(t, got)
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
					indexer.FreightByWarehouseIndexField,
					indexer.FreightByWarehouseIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightByVerifiedStagesIndexField,
					indexer.FreightByVerifiedStagesIndexer,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()
			r := &controlFlowStageReconciler{
				client: c,
			}

			got, err := r.getAvailableFreight(context.Background(), tt.stage, tt.requested)
			tt.assertions(t, got, err)
		})
	}
}

func Test_controlFlowStageReconciler_verifyFreight(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	toObjSlice := func(freights []kargoapi.Freight) []client.Object {
		ptrs := make([]client.Object, len(freights))
		for i, f := range freights {
			ptrs[i] = f.DeepCopy()
		}
		return ptrs
	}

	oneHourAgo := time.Now().Add(-time.Hour)
	oneMinuteAgo := time.Now().Add(-time.Minute)
	justNow := time.Now()

	tests := []struct {
		name       string
		stage      *kargoapi.Stage
		freight    []kargoapi.Freight
		startTime  time.Time
		finishTime time.Time
		interceptor interceptor.Funcs
		assertions func(*testing.T, client.Client, *fakeevent.EventRecorder, error)
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
			assertions: func(t *testing.T, c client.Client, recorder *fakeevent.EventRecorder, err error) {
				require.NoError(t, err)
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
				assert.Contains(t, freight2.Status.VerifiedIn, "test-stage")
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

			r := &controlFlowStageReconciler{
				client:        c,
				eventRecorder: recorder,
			}

			tt.assertions(
				t,
				c,
				recorder,
				r.verifyFreight(context.Background(), tt.stage, tt.freight, tt.startTime, tt.finishTime),
			)
		})
	}
}

func Test_controlFlowStageReconciler_handleDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name    string
		stage  *kargoapi.Stage
		interceptor interceptor.Funcs
		assertions func(*testing.T, *kargoapi.Stage, error)
	}{
		{
			name: "finalizer already removed",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-stage",
					Namespace:  "default",
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
					case strings.Contains(listOpts.FieldSelector.String(), indexer.FreightApprovedForStagesIndexField):
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
					indexer.FreightByVerifiedStagesIndexField,
					indexer.FreightByVerifiedStagesIndexer,
				).
				WithIndex(
					&kargoapi.Freight{},
					indexer.FreightApprovedForStagesIndexField,
					indexer.FreightApprovedForStagesIndexer,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &controlFlowStageReconciler{
				client:        c,
			}

			err := r.handleDelete(context.Background(), tt.stage)
			tt.assertions(t, tt.stage, err)
		})
	}
}

func Test_controlFlowStageReconciler_clearVerifications(t *testing.T) {
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
							"test-stage":     {},
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
					indexer.FreightByVerifiedStagesIndexField,
					indexer.FreightByVerifiedStagesIndexer,
				).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &controlFlowStageReconciler{
				client: c,
			}

			tt.assertions(t, c, r.clearVerifications(context.Background(), tt.stage))
		})
	}
}

func Test_controlFlowStageReconciler_clearApprovals(t *testing.T) {
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
							"test-stage":     {},
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
					indexer.FreightApprovedForStagesIndexField,
					indexer.FreightApprovedForStagesIndexer,
				).
				WithStatusSubresource(&kargoapi.Freight{}).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			r := &controlFlowStageReconciler{
				client: c,
			}

			tt.assertions(t, c, r.clearApprovals(context.Background(), tt.stage))
		})
	}
}

func Test_controlFlowStageReconciler_clearAnalysisRuns(t *testing.T) {
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
							kargoapi.StageLabelKey: "test-stage",
						},
					},
				},
				&rollouts.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "analysis-2",
						Labels: map[string]string{
							kargoapi.StageLabelKey: "test-stage",
						},
					},
				},
				&rollouts.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "analysis-other",
						Labels: map[string]string{
							kargoapi.StageLabelKey: "other-stage",
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
					client.MatchingLabels{kargoapi.StageLabelKey: "test-stage"},
				)
				require.NoError(t, err)
				assert.Empty(t, runs.Items)

				// Verify other analysis runs still exist
				err = c.List(context.Background(), &runs,
					client.InNamespace("default"),
					client.MatchingLabels{kargoapi.StageLabelKey: "other-stage"},
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
							kargoapi.StageLabelKey: "test-stage",
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

			r := &controlFlowStageReconciler{
				client: c,
				cfg:    tt.cfg,
			}

			tt.assertions(t, c, r.clearAnalysisRuns(context.Background(), tt.stage))
		})
	}
}
