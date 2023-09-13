package stages

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/akuity/kargo/api/v1alpha1"
)

// EnqueueDownstreamStagesHandler is an event handler that enqueues downstream Stages
// if a Stage's history has been modified, so that those stages can update their
// availableFreight.
type EnqueueDownstreamStagesHandler struct {
	logger      *log.Entry
	kargoClient client.Client
}

// Create implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Create(_ event.CreateEvent, _ workqueue.RateLimitingInterface) {
	// do nothing
}

// Delete implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Delete(_ event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	// do nothing
}

// Generic implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Generic(_ event.GenericEvent, _ workqueue.RateLimitingInterface) {
	// do nothing
}

// Update implements EventHandler.
func (e *EnqueueDownstreamStagesHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		e.logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	newStage, ok := evt.ObjectNew.(*v1alpha1.Stage)
	if !ok {
		e.logger.Errorf("Failed to convert new stage: %v", evt.ObjectNew)
		return
	}
	oldStage, ok := evt.ObjectOld.(*v1alpha1.Stage)
	if !ok {
		e.logger.Errorf("Failed to convert old stage: %v", evt.ObjectOld)
		return
	}
	if !newQualifiedFreight(oldStage, newStage) {
		return
	}

	// If we get here, we have new qualified freight in the Stage
	// Find downstream Stages and enqueue them
	var namespaceStages v1alpha1.StageList
	inNamespace := client.ListOptions{Namespace: newStage.Namespace}
	if err := e.kargoClient.List(context.TODO(), &namespaceStages, &inNamespace); err != nil {
		e.logger.Errorf("Failed to list downstream stages: %v", evt.ObjectOld)
		return
	}
	for _, downstreamStage := range namespaceStages.Items {
		if downstreamStage.Spec.Subscriptions == nil {
			continue
		}
		for _, upstreamStg := range downstreamStage.Spec.Subscriptions.UpstreamStages {
			if upstreamStg.Name == newStage.Name {
				q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      downstreamStage.Name,
					Namespace: downstreamStage.Namespace,
				}})
				e.logger.WithFields(log.Fields{
					"stage":     downstreamStage.Name,
					"namespace": downstreamStage.Namespace,
				}).Debug("enqueued downstream stage")
			}
		}
	}
}

// newQualifiedFreight returns whether or not there is new qualified freight in the Stage history
func newQualifiedFreight(old, new *v1alpha1.Stage) bool {
	oldQualified := make(map[string]bool)
	for _, f := range old.Status.History {
		if f.Qualified {
			oldQualified[f.ID] = true
		}
	}

	for _, f := range new.Status.History {
		if f.Qualified && !oldQualified[f.ID] {
			// something just got qualified
			return true
		}
	}
	return false
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
	promo, ok := e.Object.(*v1alpha1.Promotion)
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
	newPromo, ok := e.ObjectNew.(*v1alpha1.Promotion)
	if !ok {
		p.logger.Errorf("Failed to convert new promo: %v", e.ObjectNew)
		return false
	}
	oldPromo, ok := e.ObjectOld.(*v1alpha1.Promotion)
	if !ok {
		p.logger.Errorf("Failed to convert old promo: %v", e.ObjectOld)
		return false
	}
	if newPromo.Status.Phase.IsTerminal() && !oldPromo.Status.Phase.IsTerminal() {
		return true
	}
	return false
}
