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

// verifiedFreightEventHandler is an event handler that enqueues downstream
// Stages when Freight is marked as verified in a Stage, so that those Stages
// can reconcile and possibly create a Promotion if auto-promotion is enabled.
type verifiedFreightEventHandler struct {
	logger      *log.Entry
	kargoClient client.Client
}

// Create implements EventHandler.
func (v *verifiedFreightEventHandler) Create(
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (v *verifiedFreightEventHandler) Delete(
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (v *verifiedFreightEventHandler) Generic(
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (v *verifiedFreightEventHandler) Update(
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		v.logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	oldFreight, ok := evt.ObjectOld.(*kargoapi.Freight)
	if !ok {
		v.logger.Errorf("Failed to convert old Freight: %v", evt.ObjectOld)
		return
	}
	newFreight, ok := evt.ObjectNew.(*kargoapi.Freight)
	if !ok {
		v.logger.Errorf("Failed to convert new Freight: %v", evt.ObjectNew)
		return
	}
	newlyVerifiedStages := getNewlyVerifiedStages(oldFreight, newFreight)
	downstreamStages := map[string]struct{}{}
	for _, newlyVerifiedStage := range newlyVerifiedStages {
		stages := kargoapi.StageList{}
		if err := v.kargoClient.List(
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
			v.logger.Errorf(
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
		v.logger.WithFields(log.Fields{
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

// approvedFreightEventHandler is an event handler that enqueues Stages when
// Freight is marked as approved for them, so that those Stages can reconcile
// and possibly create a Promotion if auto-promotion is enabled.
type approvedFreightEventHandler struct {
	logger      *log.Entry
	kargoClient client.Client
}

// Create implements EventHandler.
func (a *approvedFreightEventHandler) Create(
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (a *approvedFreightEventHandler) Delete(
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (a *approvedFreightEventHandler) Generic(
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (a *approvedFreightEventHandler) Update(
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		a.logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	oldFreight, ok := evt.ObjectOld.(*kargoapi.Freight)
	if !ok {
		a.logger.Errorf("Failed to convert old Freight: %v", evt.ObjectOld)
		return
	}
	newFreight, ok := evt.ObjectNew.(*kargoapi.Freight)
	if !ok {
		a.logger.Errorf("Failed to convert new Freight: %v", evt.ObjectNew)
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
		a.logger.WithFields(log.Fields{
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

// createdFreightEventHandler is an event handler that enqueues Stages
// subscribed to a Freight's Warehouse whenever new Freight is created, so that
// those Stages can reconcile and possibly create a Promotion if auto-promotion
// is enabled.
type createdFreightEventHandler struct {
	logger      *log.Entry
	kargoClient client.Client
}

// Create implements EventHandler.
func (c *createdFreightEventHandler) Create(
	evt event.CreateEvent,
	wq workqueue.RateLimitingInterface,
) {
	freight := evt.Object.(*kargoapi.Freight) // nolint: forcetypeassert
	// TODO: Get warehouse name freight.OwnerReferences
	if len(freight.OwnerReferences) != 1 {
		c.logger.Warnf(
			"Expected Freight %q to have exactly 1 OwnerReference, got %d",
			freight.Name,
			len(freight.OwnerReferences),
		)
		return
	}
	warehouse := freight.OwnerReferences[0].Name
	stages := kargoapi.StageList{}
	if err := c.kargoClient.List(
		context.TODO(),
		&stages,
		&client.ListOptions{
			Namespace: freight.Namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.StagesByWarehouseIndexField,
				warehouse,
			),
		},
	); err != nil {
		c.logger.Errorf(
			"Failed list Stages subscribed to Warehouse %q in namespace %q",
			warehouse,
			freight.Namespace,
		)
		return
	}
	for _, stage := range stages.Items {
		wq.Add(
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: freight.Namespace,
					Name:      stage.Name,
				},
			},
		)
		c.logger.WithFields(log.Fields{
			"namespace": freight.Namespace,
			"stage":     stage.Name,
		}).Debug("enqueued Stage for reconciliation")
	}
}

// Delete implements EventHandler.
func (c *createdFreightEventHandler) Delete(
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (c *createdFreightEventHandler) Generic(
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (c *createdFreightEventHandler) Update(
	event.UpdateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}
