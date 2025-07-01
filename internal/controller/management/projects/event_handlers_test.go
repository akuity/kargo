package projects

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_projectWarehouseHealthEnqueuer_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name             string
		oldWarehouse     *kargoapi.Warehouse
		newWarehouse     *kargoapi.Warehouse
		expectedRequests []reconcile.Request
	}{
		{
			name: "no health condition change",
			oldWarehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.WarehouseStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			newWarehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.WarehouseStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionTrue,
					}},
				},
			},
		},
		{
			name: "health condition change",
			oldWarehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.WarehouseStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			newWarehouse: &kargoapi.Warehouse{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.WarehouseStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionFalse,
					}},
				},
			},
			expectedRequests: []reconcile.Request{{
				NamespacedName: types.NamespacedName{Name: "fake-project"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &projectWarehouseHealthEnqueuer[*kargoapi.Warehouse]{}
			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Update(
				context.Background(),
				event.TypedUpdateEvent[*kargoapi.Warehouse]{
					ObjectOld: tt.oldWarehouse,
					ObjectNew: tt.newWarehouse,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			require.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}

func Test_projectStageHealthEnqueuer_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	tests := []struct {
		name             string
		oldStage         *kargoapi.Stage
		newStage         *kargoapi.Stage
		expectedRequests []reconcile.Request
	}{
		{
			name: "no health condition change",
			oldStage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			newStage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionTrue,
					}},
				},
			},
		},
		{
			name: "health condition change",
			oldStage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionTrue,
					}},
				},
			},
			newStage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
				},
				Status: kargoapi.StageStatus{
					Conditions: []metav1.Condition{{
						Type:   kargoapi.ConditionTypeHealthy,
						Status: metav1.ConditionFalse,
					}},
				},
			},
			expectedRequests: []reconcile.Request{{
				NamespacedName: types.NamespacedName{Name: "fake-project"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &projectStageHealthEnqueuer[*kargoapi.Stage]{}
			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}

			enqueuer.Update(
				context.Background(),
				event.TypedUpdateEvent[*kargoapi.Stage]{
					ObjectOld: tt.oldStage,
					ObjectNew: tt.newStage,
				},
				queue,
			)

			var reqs []reconcile.Request
			for queue.Len() > 0 {
				req, _ := queue.Get()
				reqs = append(reqs, req)
				queue.Done(req)
			}

			require.ElementsMatch(t, tt.expectedRequests, reqs)
		})
	}
}
