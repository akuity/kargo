package reconciler_test

import (
	"context"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// These examples show the shapes of controller the package serves, using
// Targets. They compile but never run; the managers and connections are nil.

// TargetKey identifies a Target. The key type belongs with the resource, not
// with the engine; in the real code it sits beside the Target store.
type TargetKey struct {
	Project string
	Name    string
}

func (k TargetKey) String() string { return k.Project + "/" + k.Name }

// TargetRow is a stand-in for database.TargetRow.
type TargetRow struct {
	Project string
	Name    string
	Params  map[string]any
	Status  map[string]any
}

// targetEvents is the topic Target events travel on. The database row is
// the event body, so the API server publishes exactly what it committed. In
// the real code this is declared once beside the store.
var targetEvents = nats.NewTopic[TargetKey, TargetRow]("kargo.targets")

// targetStore is the slice of database.Store a Target controller needs.
type targetStore interface {
	GetTarget(ctx context.Context, key TargetKey) (*TargetRow, error)
	ListTargetKeys(ctx context.Context) ([]TargetKey, error)
	UpdateTargetStatus(ctx context.Context, key TargetKey, status map[string]any) error
}

// Shape 1: a database-native controller fed by NATS, Kubernetes, and a resync.
//
// The controller's request type is the Target's key. It reconciles when a
// Target is written (NATS), when a Promotion to it changes (Kubernetes), and
// on every resync (database). All three are sources pushing onto one
// channel, so a Target touched by all three at once is reconciled once.
func ExampleNew() {
	var (
		mgr   manager.Manager
		conn  *natsgo.Conn
		store targetStore
	)
	targets, err := reconciler.New[TargetKey]("targets").
		// Target writes. Deletes need no reconcile: there is nothing left.
		Watch(nats.Subject(conn, targetEvents).Ignore(nats.Deleted)).
		// Promotions, mapped to the Targets they promote to.
		Watch(kube.Keyed(
			mgr.GetCache(),
			&kargoapi.Promotion{},
			func(_ context.Context, promo *kargoapi.Promotion) []TargetKey {
				// Real code reads the Target names off the Promotion.
				return []TargetKey{{Project: promo.Namespace, Name: "example"}}
			},
			predicate.TypedResourceVersionChangedPredicate[*kargoapi.Promotion]{},
		)).
		// Boot relist and safety net for lost events. Added last so the
		// sources above are subscribed while it loads.
		Watch(list.New(store.ListTargetKeys).Every(2 * time.Minute)).
		Workers(4).
		Func(func(ctx context.Context, key TargetKey) (reconciler.Result, error) {
			target, err := store.GetTarget(ctx, key)
			if err != nil {
				// Real code treats database.ErrNotFound as done.
				return reconciler.Result{}, fmt.Errorf("error getting Target %s: %w", key, err)
			}
			// Compute per-Stage status from Promotions, then persist if it changed.
			return reconciler.Result{}, store.UpdateTargetStatus(ctx, key, target.Status)
		})
	if err != nil {
		return
	}
	// Under a manager, kube.Runnable decides leader election and the manager
	// starts it. Without one, start it directly: go targets.Start(ctx).
	_ = mgr.Add(kube.Runnable(targets))
}

// Shape 2: a plain Kubernetes controller, with nothing but Kubernetes.
//
// This is a controller-runtime For(&Stage{}).Watches(&Freight{}, ...) on
// this engine. The primary source is an informer on Stages: at startup it
// lists every Stage and pushes each one, and Start does not return until it
// has, so every Stage is reconciled once on boot. Freight changes are mapped
// to the Stages they concern. The manager starts and syncs its cache before
// it starts the controller.
func ExampleNew_kubernetesOnly() {
	var (
		mgr manager.Manager
		// stagesRequesting names the Stages that request Freight from the
		// Warehouse that produced a piece of Freight.
		stagesRequesting func(context.Context, *kargoapi.Freight) []reconcile.Request
	)
	stages, err := reconciler.New[reconcile.Request]("stage").
		// The primary kind: every Stage, listed at startup and watched after.
		Watch(kube.Kind(
			mgr.GetCache(),
			&kargoapi.Stage{},
			&handler.TypedEnqueueRequestForObject[*kargoapi.Stage]{},
			predicate.TypedGenerationChangedPredicate[*kargoapi.Stage]{},
		)).
		// A secondary kind, mapped to the Stages it affects.
		Watch(kube.Kind(
			mgr.GetCache(),
			&kargoapi.Freight{},
			handler.TypedEnqueueRequestsFromMapFunc(stagesRequesting),
		)).
		Workers(4).
		Func(func(ctx context.Context, req reconcile.Request) (reconciler.Result, error) {
			stage := &kargoapi.Stage{}
			if err := mgr.GetClient().Get(ctx, req.NamespacedName, stage); err != nil {
				// Deleted between push and reconcile: nothing to do.
				return reconciler.Result{}, client.IgnoreNotFound(err)
			}
			// Reconcile the Stage, then come back on a timer as well.
			return reconciler.Result{RequeueAfter: 5 * time.Minute}, nil
		})
	if err != nil {
		return
	}
	_ = mgr.Add(kube.Runnable(stages))
}

// Shape 3: a controller for a Kubernetes resource that also uses the database.
//
// The Project mirror reconciles Projects into the database. Its requests are
// reconcile.Request, its primary source is the Kubernetes cache, and a
// database-driven resync pushes every Project whose row is missing or
// stale. Kubernetes is one source among others, not the foundation.
func ExampleNew_kubernetesResource() {
	var (
		mgr manager.Manager
		// listStaleProjects returns the Projects whose database row disagrees
		// with Kubernetes, including rows whose Project is gone.
		listStaleProjects func(context.Context) ([]reconcile.Request, error)
	)
	projects, err := reconciler.New[reconcile.Request]("db-sync-project").
		Watch(kube.Kind(
			mgr.GetCache(),
			&kargoapi.Project{},
			&handler.TypedEnqueueRequestForObject[*kargoapi.Project]{},
			predicate.TypedResourceVersionChangedPredicate[*kargoapi.Project]{},
		)).
		Watch(list.New(listStaleProjects).Every(time.Minute)).
		Func(func(context.Context, reconcile.Request) (reconciler.Result, error) {
			// Read the Project and upsert or delete its row.
			return reconciler.Result{}, nil
		})
	if err != nil {
		return
	}
	_ = mgr.Add(kube.Runnable(projects))
}

// Shape 4: a publisher. The API server does this after each committed write.
//
// The publisher is nil when the server runs without NATS. A failure to
// publish is logged, never returned to the user: the controller's resync
// covers it.
func ExampleTopic_PublishUpdated() {
	var (
		pub           nats.Publisher
		before, after TargetRow
	)
	key := TargetKey{Project: after.Project, Name: after.Name}
	err := targetEvents.PublishUpdated(pub, key, before, after)
	_ = err
}
