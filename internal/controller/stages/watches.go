package stages

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
)

// EnqueueDownstreamStagesHandler is an event handler that enqueues downstream
// Stages when a Freight is qualified for a Stage, so that those Stages can
// reconcile and possibly create a Promotion if auto-promotion is enabled.
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
	newlyQualifiedStages := getNewlyQualifiedStages(oldFreight, newFreight)
	downstreamStages := map[string]struct{}{}
	for _, newlyQualifiedStage := range newlyQualifiedStages {
		stages := kargoapi.StageList{}
		if err := e.kargoClient.List(
			context.TODO(),
			&stages,
			&client.ListOptions{
				Namespace: newFreight.Namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.StagesByUpstreamStagesIndexField,
					newlyQualifiedStage,
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
		}).Debug("enqueued downstream stage")
	}
}

func getNewlyQualifiedStages(old, new *kargoapi.Freight) []string {
	var stages []string
	for stage := range new.Status.Qualifications {
		if _, ok := old.Status.Qualifications[stage]; !ok {
			stages = append(stages, stage)
		}
	}
	return stages
}

// PromoWentTerminal is a predicate that returns true if a promotion went terminal
type PromoWentTerminal struct {
	predicate.Funcs
	logger *log.Entry
}

func (p PromoWentTerminal) Create(_ event.CreateEvent) bool {
	return false
}

func (p PromoWentTerminal) Delete(e event.DeleteEvent) bool {
	promo, ok := e.Object.(*kargoapi.Promotion)
	// if promo is deleted but was non-terminal, we want to enqueue the
	// Stage so it can reset status.currentPromotion
	return ok && !promo.Status.Phase.IsTerminal()
}

func (p PromoWentTerminal) Generic(_ event.GenericEvent) bool {
	// we should never get here
	return true
}

// Update implements default UpdateEvent filter for checking if a promotion went terminal
func (p PromoWentTerminal) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		p.logger.Errorf("Update event has no old object to update: %v", e)
		return false
	}
	if e.ObjectNew == nil {
		p.logger.Errorf("Update event has no new object for update: %v", e)
		return false
	}
	newPromo, ok := e.ObjectNew.(*kargoapi.Promotion)
	if !ok {
		p.logger.Errorf("Failed to convert new promo: %v", e.ObjectNew)
		return false
	}
	oldPromo, ok := e.ObjectOld.(*kargoapi.Promotion)
	if !ok {
		p.logger.Errorf("Failed to convert old promo: %v", e.ObjectOld)
		return false
	}
	if newPromo.Status.Phase.IsTerminal() && !oldPromo.Status.Phase.IsTerminal() {
		return true
	}
	return false
}
