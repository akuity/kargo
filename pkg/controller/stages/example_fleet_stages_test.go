package stages

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/logging"
	intpredicate "github.com/akuity/kargo/pkg/predicate"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// These examples show FleetStageReconciler.SetupWithManager rewritten on
// pkg/reconciler. They compile but never run. Compare with the real method
// in fleet_stages.go: the indexes, handlers, predicates, and Reconcile are
// unchanged; only the assembly differs.

// The reconciler as it is today: every source is a Kubernetes kind.
//
// What changes against the controller-runtime version:
//
//   - For(&Stage{}) plus three WithEventFilter calls become one kube.Kind on
//     Stages carrying the same predicates. WithEventFilter only ever applied
//     to the For source, so nothing is lost by attaching them there.
//   - Each c.Watch(source.Kind(...)) becomes a kube.Kind(...) in Sources
//     with the arguments unchanged. The handlers in event_handlers.go
//     already implement controller-runtime's typed handler interface.
//   - The controller is built and handed to the manager, and Reconcile
//     returns reconciler.Result. The adapter below exists only so this
//     example compiles against today's signature.
func ExampleFleetStageReconciler_SetupWithManager() {
	var (
		ctx           context.Context
		kargoMgr      ctrl.Manager
		sharedIndexer client.FieldIndexer
		cfg           ReconcilerConfig
		credentialsDB credentials.Database
	)
	r := NewFleetStageReconciler(cfg, credentialsDB)
	r.client = kargoMgr.GetClient()
	logger := logging.LoggerFromContext(ctx)

	// Indexes are set up exactly as today; one is shown.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.PromotionRequest{},
		indexer.PromotionRequestsByStageField,
		indexer.PromotionRequestsByStage,
	); err != nil {
		return
	}

	sources := []reconciler.Source[reconcile.Request]{
		// The primary kind. This is For(&Stage{}) and the WithEventFilter
		// predicates, which only ever applied to it.
		kube.Kind(
			kargoMgr.GetCache(),
			client.Object(&kargoapi.Stage{}),
			&handler.EnqueueRequestForObject{},
			controller.ResponsibleFor[client.Object]{
				IsDefaultController: cfg.IsDefaultController,
				ShardName:           cfg.ShardName,
			},
			intpredicate.IgnoreDelete[client.Object]{},
			predicate.And(
				IsTargetAwareStage(true),
				IsControlFlowStage(false),
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					kargo.RefreshRequested{},
					kargo.ReverifyRequested{},
					kargo.VerificationAbortRequested{},
				),
			),
		),
		// PromotionRequests whose phase changed, mapped to the owning Stage.
		kube.Kind(
			kargoMgr.GetCache(),
			&kargoapi.PromotionRequest{},
			handler.TypedEnqueueRequestForOwner[*kargoapi.PromotionRequest](
				kargoMgr.GetScheme(),
				kargoMgr.GetRESTMapper(),
				&kargoapi.Stage{},
				handler.OnlyControllerOwner(),
			),
			kargo.NewPromotionRequestPhaseChangedPredicate(logger),
		),
		// Freight, three ways, each with the handler it has today.
		kube.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			&downstreamStageEnqueuer[*kargoapi.Freight]{kargoClient: kargoMgr.GetClient()},
		),
		kube.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			&stageEnqueuerForApprovedFreight[*kargoapi.Freight]{kargoClient: kargoMgr.GetClient()},
		),
		kube.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			&warehouseStageEnqueuer[*kargoapi.Freight]{kargoClient: kargoMgr.GetClient()},
		),
	}
	if cfg.RolloutsIntegrationEnabled {
		sources = append(sources, kube.Kind(
			kargoMgr.GetCache(),
			&rolloutsapi.AnalysisRun{},
			&stageEnqueuerForAnalysisRuns[*rolloutsapi.AnalysisRun]{kargoClient: kargoMgr.GetClient()},
		))
	}

	b := reconciler.New[reconcile.Request]("fleet_stage").Workers(cfg.MaxConcurrentFleetReconciles)
	for _, src := range sources {
		b = b.Watch(src)
	}
	c, err := b.Func(func(ctx context.Context, req reconcile.Request) (reconciler.Result, error) {
		// In a real port Reconcile returns reconciler.Result itself and this
		// adapter disappears; the body of Reconcile does not change.
		res, err := r.Reconcile(ctx, req)
		return reconciler.Result{RequeueAfter: res.RequeueAfter}, err
	})
	if err != nil {
		return
	}
	_ = kargoMgr.Add(kube.Runnable(c))
}

// promotionRequestRow stands in for the database's PromotionRequest row,
// which does not exist on this branch yet.
type promotionRequestRow struct {
	ID      uuid.UUID
	Project string
	Stage   string
	Phase   kargoapi.PromotionRequestPhase
}

// The same reconciler once PromotionRequests live in the database.
//
// Only the PromotionRequest source changes. The kube.Kind on
// PromotionRequests is replaced by a NATS subject carrying the rows the API
// server and the request reconciler commit, mapped to the Stage each one
// belongs to, with the phase-changed predicate expressed over rows. A
// periodic list of Stages with open requests covers lost events. The Stage source, the Freight sources, the indexes on
// Freight, and Reconcile are untouched. Inside Reconcile,
// syncPromotionRequests reads the requests from the store instead of the
// cache, which is a change to that function, not to the controller.
func ExampleFleetStageReconciler_SetupWithManager_databaseRequests() {
	var (
		kargoMgr ctrl.Manager
		cfg      ReconcilerConfig
		conn     *natsgo.Conn
		// listStagesWithOpenRequests returns every Stage with a non-terminal
		// PromotionRequest, from the store.
		listStagesWithOpenRequests func(context.Context) ([]reconcile.Request, error)
		stageSource                reconciler.Source[reconcile.Request]
		freightSources             []reconciler.Source[reconcile.Request]
		reconcileFn                reconciler.Func[reconcile.Request]
	)
	// Declared once beside the store in the real code.
	requestEvents := nats.NewTopic[uuid.UUID, promotionRequestRow]("kargo.promotionrequests")

	sources := []reconciler.Source[reconcile.Request]{
		// Event sources first, so they are subscribed while the resync lists.
		nats.Mapped(
			conn,
			requestEvents,
			func(_ context.Context, e nats.Event[uuid.UUID, promotionRequestRow]) []reconcile.Request {
				row := e.New
				if row == nil {
					row = e.Old
				}
				return []reconcile.Request{{
					NamespacedName: client.ObjectKey{Namespace: row.Project, Name: row.Stage},
				}}
			},
		).
			// The counterpart of kargo.PromotionRequestPhaseChanged.
			OnUpdate(func(old, updated promotionRequestRow) bool { return old.Phase != updated.Phase }),
		stageSource,
	}
	sources = append(sources, freightSources...)
	// Stages with requests in flight, in case an event was lost.
	sources = append(sources, list.New(listStagesWithOpenRequests).Every(2*time.Minute))

	b := reconciler.New[reconcile.Request]("fleet_stage").Workers(cfg.MaxConcurrentFleetReconciles)
	for _, src := range sources {
		b = b.Watch(src)
	}
	c, err := b.Func(reconcileFn)
	if err != nil {
		return
	}
	_ = kargoMgr.Add(kube.Runnable(c))
}

// targetKey and targetRow stand in for the database's Target key and row,
// which are on the Targets branches and not yet on main.
type targetKey struct {
	Project string
	Name    string
}

type targetRow struct {
	Name   string
	Labels map[string]string
	Params json.RawMessage
}

// Listening to Targets from the fleet Stage reconciler.
//
// Targets are database-native, so there is no Kubernetes kind to watch. The
// API server publishes a Target event on every committed write, and the
// Stage controller maps each event to the Stages that govern the Target: the
// target-aware Stages in the Target's Project whose selectors match its
// labels. Both the old and the new labels are checked, because a label
// change can move a Target from one Stage to another and both need to
// re-render. Updates that touch neither labels nor params are dropped, so a
// status write to the Target does not wake every Stage that governs it.
func ExampleFleetStageReconciler_SetupWithManager_targetEvents() {
	var (
		kargoMgr ctrl.Manager
		conn     *natsgo.Conn
	)
	// Declared once beside the store in the real code.
	targetEvents := nats.NewTopic[targetKey, targetRow]("kargo.targets")

	c := kargoMgr.GetClient()
	targetsToStages := nats.Mapped(
		conn,
		targetEvents,
		func(ctx context.Context, e nats.Event[targetKey, targetRow]) []reconcile.Request {
			stages := &kargoapi.StageList{}
			if err := c.List(ctx, stages, client.InNamespace(e.Key.Project)); err != nil {
				logging.LoggerFromContext(ctx).Error(err, "error listing Stages for Target event")
				return nil
			}
			var reqs []reconcile.Request
			for i := range stages.Items {
				stage := &stages.Items[i]
				selectors, err := api.TargetSelectorsForStage(stage)
				if err != nil || len(selectors) == 0 {
					continue
				}
				governedBefore := e.Old != nil && api.AnySelectorMatches(selectors, e.Old.Labels)
				governedAfter := e.New != nil && api.AnySelectorMatches(selectors, e.New.Labels)
				if governedBefore || governedAfter {
					reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(stage)})
				}
			}
			return reqs
		},
	).OnUpdate(func(old, updated targetRow) bool {
		return !maps.Equal(old.Labels, updated.Labels) || !bytes.Equal(old.Params, updated.Params)
	})

	// Added to the fleet Stage controller alongside its Kubernetes sources.
	_ = reconciler.New[reconcile.Request]("fleet_stage").Watch(targetsToStages)
}
