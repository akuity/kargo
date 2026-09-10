package projects

import (
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
				t.Context(),
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
				t.Context(),
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

func Test_projectStageHealthEnqueuer_Update_promotionRequestChanged(t *testing.T) {
	stageWith := func(current, last string) *kargoapi.Stage {
		stage := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project"},
			Status: kargoapi.StageStatus{
				Conditions: []metav1.Condition{{
					Type:   kargoapi.ConditionTypeHealthy,
					Status: metav1.ConditionTrue,
				}},
			},
		}
		if current != "" {
			stage.Status.CurrentPromotionRequest = &kargoapi.PromotionRequestReference{Name: current}
		}
		if last != "" {
			stage.Status.LastPromotionRequest = &kargoapi.PromotionRequestReference{Name: last}
		}
		return stage
	}
	project := []reconcile.Request{{NamespacedName: types.NamespacedName{Name: "fake-project"}}}

	tests := []struct {
		name             string
		oldStage         *kargoapi.Stage
		newStage         *kargoapi.Stage
		expectedRequests []reconcile.Request
	}{
		{
			name:     "same requests and same health",
			oldStage: stageWith("cur", "last"),
			newStage: stageWith("cur", "last"),
		},
		{
			name:             "current request appears",
			oldStage:         stageWith("", "last"),
			newStage:         stageWith("cur", "last"),
			expectedRequests: project,
		},
		{
			name:             "current request clears",
			oldStage:         stageWith("cur", "last"),
			newStage:         stageWith("", "last"),
			expectedRequests: project,
		},
		{
			name:             "last request moves forward",
			oldStage:         stageWith("", "old"),
			newStage:         stageWith("", "new"),
			expectedRequests: project,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &projectStageHealthEnqueuer[*kargoapi.Stage]{}
			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
			enqueuer.Update(
				t.Context(),
				event.TypedUpdateEvent[*kargoapi.Stage]{ObjectOld: tt.oldStage, ObjectNew: tt.newStage},
				queue,
			)
			require.ElementsMatch(t, tt.expectedRequests, drainQueue(queue))
		})
	}
}

func Test_projectPromotionRequestEnqueuer(t *testing.T) {
	request := func(
		phase kargoapi.PromotionRequestPhase,
		summary *kargoapi.PromotionRequestSummary,
	) *kargoapi.PromotionRequest {
		return &kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project"},
			Status:     kargoapi.PromotionRequestStatus{Phase: phase, Summary: summary},
		}
	}
	project := []reconcile.Request{{NamespacedName: types.NamespacedName{Name: "fake-project"}}}

	t.Run("create enqueues", func(t *testing.T) {
		enqueuer := &projectPromotionRequestEnqueuer[*kargoapi.PromotionRequest]{}
		queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Create(
			t.Context(),
			event.TypedCreateEvent[*kargoapi.PromotionRequest]{
				Object: request(kargoapi.PromotionRequestPhasePending, nil),
			},
			queue,
		)
		require.ElementsMatch(t, project, drainQueue(queue))
	})

	t.Run("delete enqueues", func(t *testing.T) {
		enqueuer := &projectPromotionRequestEnqueuer[*kargoapi.PromotionRequest]{}
		queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Delete(
			t.Context(),
			event.TypedDeleteEvent[*kargoapi.PromotionRequest]{
				Object: request(kargoapi.PromotionRequestPhaseSucceeded, nil),
			},
			queue,
		)
		require.ElementsMatch(t, project, drainQueue(queue))
	})

	updates := []struct {
		name             string
		oldRequest       *kargoapi.PromotionRequest
		newRequest       *kargoapi.PromotionRequest
		expectedRequests []reconcile.Request
	}{
		{
			name:       "nothing relevant changed",
			oldRequest: request(kargoapi.PromotionRequestPhaseRunning, &kargoapi.PromotionRequestSummary{Running: 2}),
			newRequest: request(kargoapi.PromotionRequestPhaseRunning, &kargoapi.PromotionRequestSummary{Running: 2}),
		},
		{
			name:             "phase changed",
			oldRequest:       request(kargoapi.PromotionRequestPhasePending, nil),
			newRequest:       request(kargoapi.PromotionRequestPhaseRunning, nil),
			expectedRequests: project,
		},
		{
			name:       "summary changed",
			oldRequest: request(kargoapi.PromotionRequestPhaseRunning, &kargoapi.PromotionRequestSummary{Running: 2}),
			newRequest: request(
				kargoapi.PromotionRequestPhaseRunning,
				&kargoapi.PromotionRequestSummary{Running: 1, Succeeded: 1},
			),
			expectedRequests: project,
		},
		{
			name:             "summary appears",
			oldRequest:       request(kargoapi.PromotionRequestPhaseRunning, nil),
			newRequest:       request(kargoapi.PromotionRequestPhaseRunning, &kargoapi.PromotionRequestSummary{Running: 2}),
			expectedRequests: project,
		},
	}
	for _, tt := range updates {
		t.Run("update: "+tt.name, func(t *testing.T) {
			enqueuer := &projectPromotionRequestEnqueuer[*kargoapi.PromotionRequest]{}
			queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
			enqueuer.Update(
				t.Context(),
				event.TypedUpdateEvent[*kargoapi.PromotionRequest]{
					ObjectOld: tt.oldRequest,
					ObjectNew: tt.newRequest,
				},
				queue,
			)
			require.ElementsMatch(t, tt.expectedRequests, drainQueue(queue))
		})
	}
}

func Test_projectTargetCountEnqueuer(t *testing.T) {
	target := &kargoapi.Target{ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project"}}
	project := []reconcile.Request{{NamespacedName: types.NamespacedName{Name: "fake-project"}}}

	t.Run("create enqueues", func(t *testing.T) {
		enqueuer := &projectTargetCountEnqueuer[*kargoapi.Target]{}
		queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Create(t.Context(), event.TypedCreateEvent[*kargoapi.Target]{Object: target}, queue)
		require.ElementsMatch(t, project, drainQueue(queue))
	})

	t.Run("delete enqueues", func(t *testing.T) {
		enqueuer := &projectTargetCountEnqueuer[*kargoapi.Target]{}
		queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Delete(t.Context(), event.TypedDeleteEvent[*kargoapi.Target]{Object: target}, queue)
		require.ElementsMatch(t, project, drainQueue(queue))
	})

	t.Run("update does not enqueue", func(t *testing.T) {
		enqueuer := &projectTargetCountEnqueuer[*kargoapi.Target]{}
		queue := &controllertest.Queue{TypedInterface: workqueue.NewTyped[reconcile.Request]()}
		enqueuer.Update(
			t.Context(),
			event.TypedUpdateEvent[*kargoapi.Target]{ObjectOld: target, ObjectNew: target.DeepCopy()},
			queue,
		)
		require.Empty(t, drainQueue(queue))
	})
}

// drainQueue returns every request on the queue, marking each done.
func drainQueue(queue *controllertest.Queue) []reconcile.Request {
	var reqs []reconcile.Request
	for queue.Len() > 0 {
		req, _ := queue.Get()
		reqs = append(reqs, req)
		queue.Done(req)
	}
	return reqs
}
