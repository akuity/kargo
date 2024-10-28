package stages

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
)

func Test_downstreamStageEnqueuer_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name             string
		oldFreight       *kargoapi.Freight
		newFreight       *kargoapi.Freight
		objects          []client.Object
		interceptor      interceptor.Funcs
		forControlFlow   bool
		expectedRequests []reconcile.Request
	}{
		{
			name: "no newly verified stages",
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"stage-1": {},
					},
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"stage-1": {},
					},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "enqueues downstream regular stages",
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"stage-1": {},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "downstream-1",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
						RequestedFreight: []kargoapi.FreightRequest{
							{
								Sources: kargoapi.FreightSources{
									Stages: []string{"stage-1"},
								},
							},
						},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "downstream-2",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
						RequestedFreight: []kargoapi.FreightRequest{
							{
								Sources: kargoapi.FreightSources{
									Stages: []string{"stage-1"},
								},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "downstream-2",
					},
				},
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "downstream-1",
					},
				},
			},
		},
		{
			name:           "enqueues downstream control flow stages when configured",
			forControlFlow: true,
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"stage-1": {},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "downstream-1",
					},
					Spec: kargoapi.StageSpec{
						RequestedFreight: []kargoapi.FreightRequest{
							{
								Sources: kargoapi.FreightSources{
									Stages: []string{"stage-1"},
								},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "downstream-1",
					},
				},
			},
		},
		{
			name: "ignores control flow stages when not configured",
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"stage-1": {},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "downstream-1",
					},
					Spec: kargoapi.StageSpec{
						RequestedFreight: []kargoapi.FreightRequest{
							{
								Sources: kargoapi.FreightSources{
									Stages: []string{"stage-1"},
								},
							},
						},
					},
				},
			},
			forControlFlow:   false,
			expectedRequests: nil,
		},
		{
			name: "list error",
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"stage-1": {},
					},
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("list error")
				},
			},
			expectedRequests: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(
					&kargoapi.Stage{},
					indexer.StagesByUpstreamStagesField,
					indexer.StagesByUpstreamStages,
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			enqueuer := &downstreamStageEnqueuer[*kargoapi.Freight]{
				kargoClient:          c,
				forControlFlowStages: tt.forControlFlow,
			}

			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Update(
				context.Background(),
				event.TypedUpdateEvent[*kargoapi.Freight]{
					ObjectOld: tt.oldFreight,
					ObjectNew: tt.newFreight,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			assert.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}

func Test_stageEnqueuerForApprovedFreight_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name             string
		oldFreight       *kargoapi.Freight
		newFreight       *kargoapi.Freight
		expectedRequests []reconcile.Request
	}{
		{
			name: "no newly approved stages",
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{
						"stage-1": {},
					},
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{
						"stage-1": {},
					},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "enqueues newly approved stages",
			oldFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
			},
			newFreight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{
						"stage-1": {},
						"stage-2": {},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "stage-1",
					},
				},
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "stage-2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			enqueuer := &stageEnqueuerForApprovedFreight[*kargoapi.Freight]{
				kargoClient: c,
			}

			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Update(
				context.Background(),
				event.TypedUpdateEvent[*kargoapi.Freight]{
					ObjectOld: tt.oldFreight,
					ObjectNew: tt.newFreight,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			assert.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}

func Test_warehouseStageEnqueuer_Create(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name             string
		freight          *kargoapi.Freight
		objects          []client.Object
		interceptor      interceptor.Funcs
		forControlFlow   bool
		expectedRequests []reconcile.Request
	}{
		{
			name: "enqueues regular stages subscribed to warehouse",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "stage-1",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
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
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "stage-2",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
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
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "stage-1",
					},
				},
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "stage-2",
					},
				},
			},
		},
		{
			name:           "ignores regular stages when configured for control flow",
			forControlFlow: true,
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "stage-1",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
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
			},
			expectedRequests: nil,
		},
		{
			name:           "enqueues control flow stages when configured",
			forControlFlow: true,
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "stage-1",
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
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "stage-1",
					},
				},
			},
		},
		{
			name: "handles list error",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("list error")
				},
			},
			expectedRequests: nil,
		},
		{
			name: "ignores non-matching warehouse",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "freight-1",
				},
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "warehouse-1",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "stage-1",
					},
					Spec: kargoapi.StageSpec{
						RequestedFreight: []kargoapi.FreightRequest{
							{
								Sources: kargoapi.FreightSources{
									Direct: true,
								},
								Origin: kargoapi.FreightOrigin{
									Kind: kargoapi.FreightOriginKindWarehouse,
									Name: "warehouse-2",
								},
							},
						},
					},
				},
			},
			expectedRequests: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(&kargoapi.Stage{}, indexer.StagesByWarehouseField, indexer.StagesByWarehouse).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			enqueuer := &warehouseStageEnqueuer[*kargoapi.Freight]{
				kargoClient:          c,
				forControlFlowStages: tt.forControlFlow,
			}

			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Create(
				context.Background(),
				event.TypedCreateEvent[*kargoapi.Freight]{
					Object: tt.freight,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			assert.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}

func Test_stageEnqueuerForArgoCDChanges_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, argocd.AddToScheme(scheme))

	tests := []struct {
		name             string
		oldApp           *argocd.Application
		newApp           *argocd.Application
		objects          []client.Object
		interceptor      interceptor.Funcs
		expectedRequests []reconcile.Request
	}{
		{
			name: "no changes in health or sync status",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "health status changed",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "test-stage",
					},
				},
			},
		},
		{
			name: "sync status changed",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "OutOfSync",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "test-stage",
					},
				},
			},
		},
		{
			name: "revision changed",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "rev1",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "rev2",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "test-stage",
					},
				},
			},
		},
		{
			name: "ignores app without stage annotation",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "ignores control flow stage",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage",
					},
					Spec: kargoapi.StageSpec{},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "handles stage not found",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "handles get error",
			oldApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			newApp: &argocd.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-app",
					Annotations: map[string]string{
						kargoapi.AnnotationKeyAuthorizedStage: "default:test-stage",
					},
				},
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return fmt.Errorf("get error")
				},
			},
			expectedRequests: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			enqueuer := &stageEnqueuerForArgoCDChanges[*argocd.Application]{
				kargoClient: c,
			}

			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Update(
				context.Background(),
				event.TypedUpdateEvent[*argocd.Application]{
					ObjectOld: tt.oldApp,
					ObjectNew: tt.newApp,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			assert.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}

func Test_stageEnqueuerForAnalysisRuns_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	require.NoError(t, rollouts.AddToScheme(scheme))

	tests := []struct {
		name             string
		oldAnalysisRun   *rollouts.AnalysisRun
		newAnalysisRun   *rollouts.AnalysisRun
		objects          []client.Object
		interceptor      interceptor.Funcs
		expectedRequests []reconcile.Request
	}{
		{
			name: "no phase change",
			oldAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Running",
				},
			},
			newAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Running",
				},
			},
			expectedRequests: nil,
		},
		{
			name: "phase changed - enqueues regular stages",
			oldAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Running",
				},
			},
			newAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Successful",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						FreightHistory: kargoapi.FreightHistory{
							{
								VerificationHistory: []kargoapi.VerificationInfo{
									{
										AnalysisRun: &kargoapi.AnalysisRunReference{
											Name:      "test-analysis",
											Namespace: "default",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "test-stage",
					},
				},
			},
		},
		{
			name: "ignores control flow stages",
			oldAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Running",
				},
			},
			newAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Successful",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage",
					},
					Status: kargoapi.StageStatus{
						FreightHistory: kargoapi.FreightHistory{
							{
								VerificationHistory: []kargoapi.VerificationInfo{
									{
										AnalysisRun: &kargoapi.AnalysisRunReference{
											Name:      "test-analysis",
											Namespace: "default",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRequests: nil,
		},
		{
			name: "handles list error",
			oldAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Running",
				},
			},
			newAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Successful",
				},
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return fmt.Errorf("list error")
				},
			},
			expectedRequests: nil,
		},
		{
			name: "handles multiple stages",
			oldAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Running",
				},
			},
			newAnalysisRun: &rollouts.AnalysisRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-analysis",
				},
				Status: rollouts.AnalysisRunStatus{
					Phase: "Successful",
				},
			},
			objects: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage-1",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						FreightHistory: kargoapi.FreightHistory{
							{
								VerificationHistory: []kargoapi.VerificationInfo{
									{
										AnalysisRun: &kargoapi.AnalysisRunReference{
											Name:      "test-analysis",
											Namespace: "default",
										},
									},
								},
							},
						},
					},
				},
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "test-stage-2",
					},
					Spec: kargoapi.StageSpec{
						PromotionTemplate: &kargoapi.PromotionTemplate{
							Spec: kargoapi.PromotionTemplateSpec{
								Steps: []kargoapi.PromotionStep{{}},
							},
						},
					},
					Status: kargoapi.StageStatus{
						FreightHistory: kargoapi.FreightHistory{
							{
								VerificationHistory: []kargoapi.VerificationInfo{
									{
										AnalysisRun: &kargoapi.AnalysisRunReference{
											Name:      "test-analysis",
											Namespace: "default",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedRequests: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "test-stage-1",
					},
				},
				{
					NamespacedName: types.NamespacedName{
						Namespace: "default",
						Name:      "test-stage-2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				WithIndex(
					&kargoapi.Stage{},
					indexer.StagesByAnalysisRunField,
					indexer.StagesByAnalysisRun(""),
				).
				WithInterceptorFuncs(tt.interceptor).
				Build()

			enqueuer := &stageEnqueuerForAnalysisRuns[*rollouts.AnalysisRun]{
				kargoClient: c,
			}

			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Update(
				context.Background(),
				event.TypedUpdateEvent[*rollouts.AnalysisRun]{
					ObjectOld: tt.oldAnalysisRun,
					ObjectNew: tt.newAnalysisRun,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			assert.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}

func Test_appHealthOrSyncStatusChanged(t *testing.T) {
	testCases := []struct {
		name    string
		old     *argocd.Application
		new     *argocd.Application
		updated bool
	}{
		{
			name: "health changed",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Degraded",
					},
				},
			},
			updated: true,
		},
		{
			name: "health did not change",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Health: argocd.HealthStatus{
						Status: "Healthy",
					},
				},
			},
			updated: false,
		},
		{
			name: "sync status changed",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			updated: true,
		},
		{
			name: "sync status did not change",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Status: "Synced",
					},
				},
			},
			updated: false,
		},
		{
			name: "revision changed",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "different-fake-revision",
					},
				},
			},
			updated: true,
		},
		{
			name: "revision did not change",
			old: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			new: &argocd.Application{
				Status: argocd.ApplicationStatus{
					Sync: argocd.SyncStatus{
						Revision: "fake-revision",
					},
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := event.UpdateEvent{
				ObjectOld: testCase.old,
				ObjectNew: testCase.new,
			}
			require.Equal(
				t,
				testCase.updated,
				appHealthOrSyncStatusChanged(context.Background(), e),
			)
		})
	}
}

func Test_analysisRunPhaseChanged(t *testing.T) {
	testCases := []struct {
		name    string
		old     *rollouts.AnalysisRun
		new     *rollouts.AnalysisRun
		updated bool
	}{
		{
			name: "phase changed",
			old: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "old-phase",
				},
			},
			new: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "new-phase",
				},
			},
			updated: true,
		},
		{
			name: "phase did not change",
			old: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "old-phase",
				},
			},
			new: &rollouts.AnalysisRun{
				Status: rollouts.AnalysisRunStatus{
					Phase: "old-phase",
				},
			},
			updated: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			e := event.UpdateEvent{
				ObjectOld: testCase.old,
				ObjectNew: testCase.new,
			}
			require.Equal(
				t,
				testCase.updated,
				analysisRunPhaseChanged(context.Background(), e),
			)
		})
	}
}
