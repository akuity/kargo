package stages

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// verifiedFreightEventHandler is an event handler that enqueues downstream
// Stages when Freight is marked as verified in a Stage, so that those Stages
// can reconcile and possibly create a Promotion if auto-promotion is enabled.
type verifiedFreightEventHandler struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements EventHandler.
func (v *verifiedFreightEventHandler) Create(
	context.Context,
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (v *verifiedFreightEventHandler) Delete(
	context.Context,
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (v *verifiedFreightEventHandler) Generic(
	context.Context,
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (v *verifiedFreightEventHandler) Update(
	ctx context.Context,
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	logger := logging.LoggerFromContext(ctx)
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	oldFreight, ok := evt.ObjectOld.(*kargoapi.Freight)
	if !ok {
		logger.Errorf("Failed to convert old Freight: %v", evt.ObjectOld)
		return
	}
	newFreight, ok := evt.ObjectNew.(*kargoapi.Freight)
	if !ok {
		logger.Errorf("Failed to convert new Freight: %v", evt.ObjectNew)
		return
	}
	newlyVerifiedStages := getNewlyVerifiedStages(oldFreight, newFreight)
	downstreamStages := map[string]struct{}{}
	for _, newlyVerifiedStage := range newlyVerifiedStages {
		stages := kargoapi.StageList{}
		if err := v.kargoClient.List(
			ctx,
			&stages,
			&client.ListOptions{
				Namespace: newFreight.Namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.StagesByUpstreamStagesIndexField,
					newlyVerifiedStage,
				),
				LabelSelector: v.shardSelector,
			},
		); err != nil {
			logger.Errorf(
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
		logger.WithFields(log.Fields{
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
	kargoClient client.Client
}

// Create implements EventHandler.
func (a *approvedFreightEventHandler) Create(
	context.Context,
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (a *approvedFreightEventHandler) Delete(
	context.Context,
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (a *approvedFreightEventHandler) Generic(
	context.Context,
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (a *approvedFreightEventHandler) Update(
	ctx context.Context,
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	logger := logging.LoggerFromContext(ctx)
	if evt.ObjectOld == nil || evt.ObjectNew == nil {
		logger.Errorf("Update event has no old or new object to update: %v", evt)
		return
	}
	oldFreight, ok := evt.ObjectOld.(*kargoapi.Freight)
	if !ok {
		logger.Errorf("Failed to convert old Freight: %v", evt.ObjectOld)
		return
	}
	newFreight, ok := evt.ObjectNew.(*kargoapi.Freight)
	if !ok {
		logger.Errorf("Failed to convert new Freight: %v", evt.ObjectNew)
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
		logger.WithFields(log.Fields{
			"namespace": newFreight.Namespace,
			"stage":     stage,
		}).Debug("enqueued Stage for reconciliation")
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
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements EventHandler.
func (c *createdFreightEventHandler) Create(
	ctx context.Context,
	evt event.CreateEvent,
	wq workqueue.RateLimitingInterface,
) {
	logger := logging.LoggerFromContext(ctx)
	freight := evt.Object.(*kargoapi.Freight) // nolint: forcetypeassert
	// TODO: Get warehouse name freight.OwnerReferences
	if len(freight.OwnerReferences) != 1 {
		logger.Warnf(
			"Expected Freight %q to have exactly 1 OwnerReference, got %d",
			freight.Name,
			len(freight.OwnerReferences),
		)
		return
	}
	warehouse := freight.OwnerReferences[0].Name
	stages := kargoapi.StageList{}
	if err := c.kargoClient.List(
		ctx,
		&stages,
		&client.ListOptions{
			Namespace: freight.Namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.StagesByWarehouseIndexField,
				warehouse,
			),
			LabelSelector: c.shardSelector,
		},
	); err != nil {
		logger.Errorf(
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
		logger.WithFields(log.Fields{
			"namespace": freight.Namespace,
			"stage":     stage.Name,
		}).Debug("enqueued Stage for reconciliation")
	}
}

// Delete implements EventHandler.
func (c *createdFreightEventHandler) Delete(
	context.Context,
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (c *createdFreightEventHandler) Generic(
	context.Context,
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (c *createdFreightEventHandler) Update(
	context.Context,
	event.UpdateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// updatedArgoCDAppHandler is an event handler that enqueues Stages associated
// with an Argo CD Application whenever that Application's health or sync status
// changes, so that those Stages can reconcile.
type updatedArgoCDAppHandler struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements EventHandler.
func (u *updatedArgoCDAppHandler) Create(
	context.Context,
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (u *updatedArgoCDAppHandler) Delete(
	context.Context,
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (u *updatedArgoCDAppHandler) Generic(
	context.Context,
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (u *updatedArgoCDAppHandler) Update(
	ctx context.Context,
	e event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if appHealthOrSyncStatusChanged(ctx, e) {
		logger := logging.LoggerFromContext(ctx)
		stages := &kargoapi.StageList{}
		if err := u.kargoClient.List(
			ctx,
			stages,
			&client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.StagesByArgoCDApplicationsIndexField,
					fmt.Sprintf(
						"%s:%s",
						e.ObjectNew.GetNamespace(),
						e.ObjectNew.GetName(),
					),
				),
				LabelSelector: u.shardSelector,
			},
		); err != nil {
			logger.Errorf(
				"error listing Stages for Application %q in namespace %q: %s",
				e.ObjectNew.GetNamespace(),
				e.ObjectNew.GetName(),
				err,
			)
		}
		for _, stage := range stages.Items {
			wq.Add(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: stage.Namespace,
						Name:      stage.Name,
					},
				},
			)
			logger.WithFields(log.Fields{
				"namespace": stage.Namespace,
				"stage":     stage.Name,
				"app":       e.ObjectNew.GetName(),
			}).Debug("enqueued Stage for reconciliation")
		}
	}
}

func appHealthOrSyncStatusChanged(ctx context.Context, e event.UpdateEvent) bool {
	logger := logging.LoggerFromContext(ctx)
	if e.ObjectOld == nil {
		logger.Errorf("Update event has no old object to update: %v", e)
	}
	if e.ObjectNew == nil {
		logger.Errorf("Update event has no new object for update: %v", e)
	}
	newUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectNew)
	if err != nil {
		logger.Errorf("Failed to convert new app: %v", e.ObjectNew)
	}
	oldUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectOld)
	if err != nil {
		logger.Errorf("Failed to convert old app: %v", e.ObjectOld)
	}
	oldHealth, _, _ := unstructured.NestedString(oldUn, "status", "health", "status")
	newHealth, _, _ := unstructured.NestedString(newUn, "status", "health", "status")
	// TODO: switch from checking sync status to whether or not operation is complete
	oldSync, _, _ := unstructured.NestedString(oldUn, "status", "sync", "status")
	newSync, _, _ := unstructured.NestedString(newUn, "status", "sync", "status")
	//_, oldOp := oldUn["operation"]
	//_, newOp := newUn["operation"]
	oldRev, _, _ := unstructured.NestedString(oldUn, "status", "sync", "revision")
	newRev, _, _ := unstructured.NestedString(newUn, "status", "sync", "revision")
	return newHealth != oldHealth || oldSync != newSync || oldRev != newRev
}

// phaseChangedAnalysisRunHandler is an event handler that enqueues Stages
// associated with an Argo Rollouts AnalysisRun whenever that AnalysisRun's
// phase changes.
type phaseChangedAnalysisRunHandler struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements EventHandler.
func (p *phaseChangedAnalysisRunHandler) Create(
	context.Context,
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler.
func (p *phaseChangedAnalysisRunHandler) Delete(
	context.Context,
	event.DeleteEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Generic implements EventHandler.
func (p *phaseChangedAnalysisRunHandler) Generic(
	context.Context,
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler.
func (p *phaseChangedAnalysisRunHandler) Update(
	ctx context.Context,
	e event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if analysisRunPhaseChanged(ctx, e) {
		logger := logging.LoggerFromContext(ctx)
		// Find the Stage associated with this AnalysisRun
		stages := &kargoapi.StageList{}
		if err := p.kargoClient.List(
			ctx,
			stages,
			&client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.StagesByAnalysisRunIndexField,
					fmt.Sprintf(
						"%s:%s",
						e.ObjectNew.GetNamespace(),
						e.ObjectNew.GetName(),
					),
				),
				LabelSelector: p.shardSelector,
			},
		); err != nil {
			logger.Errorf(
				"error listing Stages for AnalysisRun %q in namespace %q: %s",
				e.ObjectNew.GetNamespace(),
				e.ObjectNew.GetName(),
				err,
			)
		}
		for _, stage := range stages.Items {
			wq.Add(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: stage.Namespace,
						Name:      stage.Name,
					},
				},
			)
			logger.WithFields(log.Fields{
				"namespace":   stage.Namespace,
				"stage":       stage.Name,
				"analysisRun": e.ObjectNew.GetName(),
			}).Debug("enqueued Stage for reconciliation")
		}
	}
}

func analysisRunPhaseChanged(ctx context.Context, e event.UpdateEvent) bool {
	logger := logging.LoggerFromContext(ctx)
	if e.ObjectOld == nil {
		logger.Errorf("Update event has no old object to update: %v", e)
	}
	if e.ObjectNew == nil {
		logger.Errorf("Update event has no new object for update: %v", e)
	}
	newUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectNew)
	if err != nil {
		logger.Errorf("Failed to convert new AnalysisRun: %v", e.ObjectNew)
	}
	oldUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectOld)
	if err != nil {
		logger.Errorf("Failed to convert old AnalysisRun: %v", e.ObjectOld)
	}
	oldPhase, _, _ := unstructured.NestedString(oldUn, "status", "phase")
	newPhase, _, _ := unstructured.NestedString(newUn, "status", "phase")
	return newPhase != oldPhase
}
