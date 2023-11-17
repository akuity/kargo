package stages

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
)

// EnqueueDownstreamStagesHandler is an event handler that enqueues downstream
// Stages when a Freight is marked as verified in a Stage, so that those Stages
// can reconcile and possibly create a Promotion if auto-promotion is enabled.
type EnqueueDownstreamStagesHandler struct {
	logger      *log.Entry
	kargoClient client.Client
}

// Create implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Create(
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Delete(
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Generic(
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Update(
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		e.logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	oldFreight, ok := evt.ObjectOld.(*kargoapi.Freight)
	if !ok {
		e.logger.Errorf("Failed to convert old Freight: %v", evt.ObjectOld)
		return
	}
	newFreight, ok := evt.ObjectNew.(*kargoapi.Freight)
	if !ok {
		e.logger.Errorf("Failed to convert new Freight: %v", evt.ObjectNew)
		return
	}
	newlyVerifiedStages := getNewlyVerifiedStages(oldFreight, newFreight)
	downstreamStages := map[string]struct{}{}
	for _, newlyVerifiedStage := range newlyVerifiedStages {
		stages := kargoapi.StageList{}
		if err := e.kargoClient.List(
			context.TODO(),
			&stages,
			&client.ListOptions{
				Namespace: newFreight.Namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.StagesByUpstreamStagesIndexField,
					newlyVerifiedStage,
				),
			},
		); err != nil {
			e.logger.Errorf(
				"Failed list Stages downstream from Stage %v in namespace %q",
				evt.ObjectOld,
				newFreight.Namespace,
			)
			return
		}
		for _, stage := range stages.Items {
			downstreamStages[stage.Name] = struct{}{}
		}
	}
	for downStreamStage := range downstreamStages {
		wq.Add(
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: newFreight.Namespace,
					Name:      downStreamStage,
				},
			},
		)
		e.logger.WithFields(log.Fields{
			"namespace": newFreight.Namespace,
			"stage":     downStreamStage,
		}).Debug("enqueued downstream Stage for reconciliation")
	}
}

func getNewlyVerifiedStages(old, new *kargoapi.Freight) []string {
	var stages []string
	for stage := range new.Status.VerifiedIn {
		if _, ok := old.Status.VerifiedIn[stage]; !ok {
			stages = append(stages, stage)
		}
	}
	return stages
}

// EnqueueApprovedStagesHandler is an event handler that enqueues Stages when
// Freight is marked as approved for them, so that those Stages can reconcile
// and possibly create a Promotion if auto-promotion is enabled.
type EnqueueApprovedStagesHandler struct {
	logger      *log.Entry
	kargoClient client.Client
}

// Create implements EventHandler.
func (e *EnqueueApprovedStagesHandler) Create(
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (e *EnqueueApprovedStagesHandler) Delete(
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (e *EnqueueApprovedStagesHandler) Generic(
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (e *EnqueueApprovedStagesHandler) Update(
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		e.logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	oldFreight, ok := evt.ObjectOld.(*kargoapi.Freight)
	if !ok {
		e.logger.Errorf("Failed to convert old Freight: %v", evt.ObjectOld)
		return
	}
	newFreight, ok := evt.ObjectNew.(*kargoapi.Freight)
	if !ok {
		e.logger.Errorf("Failed to convert new Freight: %v", evt.ObjectNew)
		return
	}
	newlyApprovedStages := getNewlyApprovedStages(oldFreight, newFreight)
	for _, stage := range newlyApprovedStages {
		wq.Add(
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: newFreight.Namespace,
					Name:      stage,
				},
			},
		)
		e.logger.WithFields(log.Fields{
			"namespace": newFreight.Namespace,
			"stage":     stage,
		}).Debug("enqueued Stage fir reconciliation")
	}
}

func getNewlyApprovedStages(old, new *kargoapi.Freight) []string {
	var stages []string
	for stage := range new.Status.ApprovedFor {
		if _, ok := old.Status.ApprovedFor[stage]; !ok {
			stages = append(stages, stage)
		}
	}
	return stages
}
