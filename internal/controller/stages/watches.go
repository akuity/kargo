package stages

import (
	"context"
	"fmt"
	"strings"

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
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// verifiedFreightEventHandler is an event handler that enqueues downstream
// Stages when Freight is marked as verified in a Stage, so that those Stages
// can reconcile and possibly create a Promotion if auto-promotion is enabled.
type verifiedFreightEventHandler[T any] struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements TypedEventHandler.
func (v *verifiedFreightEventHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (v *verifiedFreightEventHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (v *verifiedFreightEventHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (v *verifiedFreightEventHandler[T]) Update(
	ctx context.Context,
	evt event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)
	oldFreight := any(evt.ObjectOld).(*kargoapi.Freight) // nolint: forcetypeassert
	newFreight := any(evt.ObjectNew).(*kargoapi.Freight) // nolint: forcetypeassert
	if oldFreight == nil || newFreight == nil {
		logger.Error(
			nil, "Update event has no old or new object to update",
			"event", evt,
		)
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
			logger.Error(
				err, "Failed to list downstream Stages",
				"stage", evt.ObjectOld,
				"namespace", newFreight.Namespace,
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
		logger.Debug(
			"enqueued downstream Stage for reconciliation",
			"namespace", newFreight.Namespace,
			"stage", downStreamStage,
		)
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
type approvedFreightEventHandler[T any] struct {
	kargoClient client.Client
}

// Create implements TypedEventHandler.
func (a *approvedFreightEventHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (a *approvedFreightEventHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (a *approvedFreightEventHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (a *approvedFreightEventHandler[T]) Update(
	ctx context.Context,
	evt event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)
	oldFreight := any(evt.ObjectOld).(*kargoapi.Freight) // nolint: forcetypeassert
	newFreight := any(evt.ObjectNew).(*kargoapi.Freight) // nolint: forcetypeassert
	if oldFreight == nil || newFreight == nil {
		logger.Error(
			nil, "Update event has no old or new object to update",
			"event", evt,
		)
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
		logger.Debug(
			"enqueued Stage for reconciliation",
			"namespace", newFreight.Namespace,
			"stage", stage,
		)
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
type createdFreightEventHandler[T any] struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements TypedEventHandler.
func (c *createdFreightEventHandler[T]) Create(
	ctx context.Context,
	evt event.TypedCreateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)
	freight := any(evt.Object).(*kargoapi.Freight) // nolint: forcetypeassert
	stages := kargoapi.StageList{}
	if err := c.kargoClient.List(
		ctx,
		&stages,
		&client.ListOptions{
			Namespace: freight.Namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.StagesByWarehouseIndexField,
				freight.Origin.Name,
			),
			LabelSelector: c.shardSelector,
		},
	); err != nil {
		logger.Error(
			err, "Failed to list Stages subscribed to Warehouse",
			"warehouse", freight.Origin.Name,
			"namespace", freight.Namespace,
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
		logger.Debug(
			"enqueued Stage for reconciliation",
			"namespace", freight.Namespace,
			"stage", stage.Name,
		)
	}
}

// Delete implements TypedEventHandler.
func (c *createdFreightEventHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (c *createdFreightEventHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements EventHandler.
func (c *createdFreightEventHandler[T]) Update(
	context.Context,
	event.TypedUpdateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// updatedArgoCDAppHandler is an event handler that enqueues Stages associated
// with an Argo CD Application whenever that Application's health or sync status
// changes, so that those Stages can reconcile.
type updatedArgoCDAppHandler[T any] struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements TypedEventHandler.
func (u *updatedArgoCDAppHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (u *updatedArgoCDAppHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (u *updatedArgoCDAppHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (u *updatedArgoCDAppHandler[T]) Update(
	ctx context.Context,
	e event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if appHealthOrSyncStatusChanged(ctx, e) {
		newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert

		stageRef, ok := newApp.Annotations[kargoapi.AnnotationKeyAuthorizedStage]
		if !ok {
			return
		}
		parts := strings.SplitN(stageRef, ":", 2)
		if len(parts) != 2 {
			return
		}
		projectName, stageName := parts[0], parts[1]

		logger := logging.LoggerFromContext(ctx)
		stage := &kargoapi.Stage{}
		if err := u.kargoClient.Get(
			ctx,
			types.NamespacedName{
				Namespace: projectName,
				Name:      stageName,
			},
			stage,
		); err != nil {
			if client.IgnoreNotFound(err) != nil {
				logger.Error(
					err,
					"error getting Stage for Application",
					"namespace", projectName,
					"stage", stageName,
					"app", newApp.Name,
				)
			}
			return
		}

		if u.shardSelector.Empty() || u.shardSelector.Matches(labels.Set(stage.Labels)) {
			wq.Add(
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: stage.Namespace,
						Name:      stage.Name,
					},
				},
			)
			logger.Debug(
				"enqueued Stage for reconciliation",
				"namespace", stage.Namespace,
				"stage", stage.Name,
				"app", newApp.Name,
			)
		}
	}
}

func appHealthOrSyncStatusChanged[T any](ctx context.Context, e event.TypedUpdateEvent[T]) bool {
	logger := logging.LoggerFromContext(ctx)
	oldApp := any(e.ObjectOld).(*argocd.Application) // nolint: forcetypeassert
	if oldApp == nil {
		logger.Error(
			nil, "Update event has no old object to update",
			"event", e,
		)
	}
	newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert
	if newApp == nil {
		logger.Error(
			nil, "Update event has no new object for update",
			"event", e,
		)
	}
	newUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(newApp)
	if err != nil {
		logger.Error(
			err, "Failed to convert new app",
			"app", newApp,
		)
	}
	oldUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oldApp)
	if err != nil {
		logger.Error(
			err, "Failed to convert old app",
			"app", oldApp,
		)
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
type phaseChangedAnalysisRunHandler[T any] struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements TypedEventHandler.
func (p *phaseChangedAnalysisRunHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (p *phaseChangedAnalysisRunHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (p *phaseChangedAnalysisRunHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (p *phaseChangedAnalysisRunHandler[T]) Update(
	ctx context.Context,
	e event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if analysisRunPhaseChanged(ctx, e) {
		analysisRun := any(e.ObjectNew).(*rollouts.AnalysisRun) // nolint: forcetypeassert
		logger := logging.LoggerFromContext(ctx)
		// Find the Stage associated with this AnalysisRun
		stages := &kargoapi.StageList{}
		if err := p.kargoClient.List(
			ctx,
			stages,
			&client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.StagesByAnalysisRunIndexField,
					fmt.Sprintf("%s:%s", analysisRun.Namespace, analysisRun.Name),
				),
				LabelSelector: p.shardSelector,
			},
		); err != nil {
			logger.Error(
				err, "error listing Stages for AnalysisRun",
				"analysisRun", analysisRun.Name,
				"namespace", analysisRun.Namespace,
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
			logger.Debug(
				"enqueued Stage for reconciliation",
				"namespace", stage.Namespace,
				"stage", stage.Name,
				"analysisRun", analysisRun.Name,
			)
		}
	}
}

func analysisRunPhaseChanged[T any](ctx context.Context, e event.TypedUpdateEvent[T]) bool {
	logger := logging.LoggerFromContext(ctx)
	oldApp := any(e.ObjectOld).(*rollouts.AnalysisRun) // nolint: forcetypeassert
	if oldApp == nil {
		logger.Error(
			nil, "Update event has no old object to update",
			"event", e,
		)
	}
	newApp := any(e.ObjectNew).(*rollouts.AnalysisRun) // nolint: forcetypeassert
	if newApp == nil {
		logger.Error(
			nil, "Update event has no new object for update",
			"event", e,
		)
	}
	newUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectNew)
	if err != nil {
		logger.Error(
			err, "Failed to convert new object to AnalysisRun",
			"object", e.ObjectNew,
		)
	}
	oldUn, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.ObjectOld)
	if err != nil {
		logger.Error(
			err, "Failed to convert old object to AnalysisRun",
			"object", e.ObjectOld,
		)
	}
	oldPhase, _, _ := unstructured.NestedString(oldUn, "status", "phase")
	newPhase, _, _ := unstructured.NestedString(newUn, "status", "phase")
	return newPhase != oldPhase
}
