package stages

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

// stageEnqueuerForArgoCDChanges triggers reconciliation of Stages when their
// associated Argo CD Application's health or sync status changes.
// The associated Stage is determined by the Application's annotations.
type stageEnqueuerForArgoCDChanges[T any] struct {
	kargoClient client.Client
}

// Create implements TypedEventHandler.
func (u *stageEnqueuerForArgoCDChanges[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (u *stageEnqueuerForArgoCDChanges[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (u *stageEnqueuerForArgoCDChanges[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (u *stageEnqueuerForArgoCDChanges[T]) Update(
	ctx context.Context,
	e event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if !appHealthOrSyncStatusChanged(ctx, e) {
		return
	}

	newApp, ok := any(e.ObjectNew).(*argocd.Application)
	if !ok {
		return
	}

	stageRef, ok := newApp.Annotations[kargoapi.AnnotationKeyAuthorizedStage]
	if !ok {
		return
	}

	logger := logging.LoggerFromContext(ctx)
	authorizedStages, err := api.AuthorizedStages(stageRef)
	if err != nil {
		logger.Error(
			err,
			"error parsing authorized Stages for Application",
			"app", newApp.Name,
			"namespace", newApp.Namespace,
		)
		return
	}

	for _, ref := range authorizedStages {
		stage := &kargoapi.Stage{}
		if err := u.kargoClient.Get(ctx, ref, stage); err != nil {
			if client.IgnoreNotFound(err) != nil {
				logger.Error(
					err,
					"error getting Stage for Application",
					"namespace", ref.Namespace,
					"stage", ref.Name,
					"app", newApp.Name,
				)
			}
			continue
		}

		// If the Stage is a control flow Stage, there is no need to reconcile it.
		if stage.IsControlFlow() {
			continue
		}

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
	// The Kargo health checker for Argo CD Applications evaluates several
	// fields from the Application status. We watch for changes to each of
	// these so that Stages are re-enqueued promptly when the health
	// checker's output might change.

	// 1. Health status: The health checker maps this directly to Stage
	//    health (Progressing, Healthy, Degraded, etc.). A change here
	//    almost always means the health checker will produce a different
	//    result.
	oldHealth, _, _ := unstructured.NestedString(
		oldUn, "status", "health", "status",
	)
	newHealth, _, _ := unstructured.NestedString(
		newUn, "status", "health", "status",
	)
	if newHealth != oldHealth {
		return true
	}

	// 2. Operation phase: The health checker returns Unknown whenever the
	//    last operation has not succeeded (e.g. it is still Running or has
	//    Failed). Watching for phase changes lets us detect both operation
	//    start (health becomes Unknown) and completion (health can be
	//    reassessed).
	oldPhase, _, _ := unstructured.NestedString(
		oldUn, "status", "operationState", "phase",
	)
	newPhase, _, _ := unstructured.NestedString(
		newUn, "status", "operationState", "phase",
	)
	if oldPhase != newPhase {
		return true
	}

	// 3. Sync revision(s): The health checker compares observed revisions
	//    against the desired revisions from the last Promotion. If the
	//    Application syncs to a new revision, the comparison result may
	//    change. We check both the single-source revision field and the
	//    multi-source revisions array.
	oldRev, _, _ := unstructured.NestedString(
		oldUn, "status", "sync", "revision",
	)
	newRev, _, _ := unstructured.NestedString(
		newUn, "status", "sync", "revision",
	)
	if oldRev != newRev {
		return true
	}
	oldRevs, _, _ := unstructured.NestedSlice(
		oldUn, "status", "sync", "revisions",
	)
	newRevs, _, _ := unstructured.NestedSlice(
		newUn, "status", "sync", "revisions",
	)
	if fmt.Sprint(oldRevs) != fmt.Sprint(newRevs) {
		return true
	}

	// 4. reconciledAt vs finishedAt: Argo CD uses separate reconciliation
	//    loops for syncing (operations) and assessing health. After an
	//    operation completes, the health status may be stale -- still
	//    reflecting pre-operation state. The health checker handles this
	//    by not trusting health until reconciledAt (set by Argo CD's
	//    health/status loop) is after finishedAt (set when the operation
	//    completed).
	//
	//    The checks above (health, revision) catch cases where the fresh
	//    post-operation values differ from the stale ones. But when the
	//    fresh values are the SAME as the stale values (e.g. the app was
	//    Healthy before the sync and is still Healthy after), those checks
	//    see no diff and don't fire. Yet the health checker's output DOES
	//    change: from Unknown (untrusted) to the now-trusted value.
	//
	//    To handle this, we check whether reconciledAt just crossed the
	//    finishedAt threshold. We only trigger when the OLD reconciledAt
	//    was stale (absent or before finishedAt) and has now advanced. Once
	//    reconciledAt is at or past finishedAt, subsequent advances are
	//    routine Argo CD refreshes and do not need to trigger
	//    re-reconciliation.
	oldReconciledAt, _, _ := unstructured.NestedString(
		oldUn, "status", "reconciledAt",
	)
	newReconciledAt, _, _ := unstructured.NestedString(
		newUn, "status", "reconciledAt",
	)
	if oldReconciledAt == newReconciledAt {
		return false
	}
	finishedAt, _, _ := unstructured.NestedString(
		newUn, "status", "operationState", "finishedAt",
	)
	if finishedAt == "" {
		return false
	}
	// RFC3339 timestamps are lexicographically orderable, so string
	// comparison preserves chronological order.
	return oldReconciledAt == "" || oldReconciledAt < finishedAt
}

func NewStageEnqueuerForAnalysisRuns[T any](
	kargoClient client.Client,
) handler.TypedEventHandler[T, reconcile.Request] {
	return &stageEnqueuerForAnalysisRuns[T]{
		kargoClient,
	}
}

// stageEnqueuerForAnalysisRuns triggers reconciliation of Stages when their
// associated Argo Rollouts AnalysisRun's phase changes.
type stageEnqueuerForAnalysisRuns[T any] struct {
	kargoClient client.Client
}

// Create implements TypedEventHandler.
func (p *stageEnqueuerForAnalysisRuns[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (p *stageEnqueuerForAnalysisRuns[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (p *stageEnqueuerForAnalysisRuns[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (p *stageEnqueuerForAnalysisRuns[T]) Update(
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
					indexer.StagesByAnalysisRunField,
					fmt.Sprintf("%s:%s", analysisRun.Namespace, analysisRun.Name),
				),
			},
		); err != nil {
			logger.Error(
				err, "error listing Stages for AnalysisRun",
				"analysisRun", analysisRun.Name,
				"namespace", analysisRun.Namespace,
			)
		}
		for _, stage := range stages.Items {
			// If the Stage is a control flow Stage, there is no need to reconcile it.
			if stage.IsControlFlow() {
				continue
			}

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
