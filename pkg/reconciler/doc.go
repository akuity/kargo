// Package reconciler is a small engine for level-triggered controllers whose
// resources live anywhere: in the database, in Kubernetes, or in both.
//
// A controller is a set of watchers, a queue, and a Reconciler. Each watcher
// pushes typed requests onto a channel the controller hands it. The
// controller moves them into a work queue and workers take them off the
// queue and hand each to the Reconciler:
//
//	Source ─┐
//	Source ─┼─► chan ─► work queue ─► workers ─► Reconciler
//	Source ─┘
//
// The queue deduplicates: however many watchers push a request while it
// waits or is being reconciled, it is reconciled once more, not once per
// push. No request is reconciled by two workers at once. A Reconcile that
// fails is retried with per-request exponential backoff, and one that
// returns a Result with RequeueAfter runs again after that long.
//
// A request carries only a key. Reconcile reads the resource fresh and acts
// on what it finds, so it never acts on a stale copy and never needs to know
// which watcher caused it to run. A push means "look at this again", not
// "this is what changed". That is what makes a controller level-triggered.
//
// # Sources
//
// A Source is something that can be watched for. Its Start is handed the
// channel and begins pushing. A source with initial state, such as the
// objects an informer lists or the rows a store holds, pushes all of it
// before returning, so that the workers begin with a complete picture; one
// that cannot returns an error and the controller does not run. That is the
// whole contract, and this package defines no sources. Each is an
// implementation in a subpackage:
//
//   - Package list pushes whatever a function returns, on start and then on
//     an interval. A store hands it the function that lists its rows, and
//     that is both the boot relist and the safety net under a lossy event
//     feed.
//   - Package nats pushes requests for the events published to a NATS
//     topic, filtered and mapped by the caller. It also publishes them.
//   - Package kube pushes requests for changes to Kubernetes objects of a
//     kind, from a controller-runtime cache, using controller-runtime's own
//     handlers and predicates so existing ones carry over.
//
// Sources start one at a time, in order. A controller lists event sources
// first, so they are subscribed while a listing source loads, and a write
// that lands during the load is caught; the queue deduplicates it against
// the load, so each resource is still reconciled once.
//
// # Requests
//
// A request is whatever key the Reconciler needs to look its resource up,
// and a controller picks the type: a Project-and-name struct for Targets,
// uuid.UUID for rows addressed by ID, reconcile.Request for Kubernetes
// objects. The key type belongs with the resource, not with this package,
// which knows nothing about what it reconciles. It must be comparable, since
// the queue deduplicates by it, so it holds identifiers, never the resource. Every source maps what it
// observes to the controller's one request type, so a resource that a NATS
// event, a Promotion update and a resync all touch at once is reconciled
// once.
//
// # Running
//
// A Controller is started with Start and runs until its context is done.
// That is all the core knows about running. A process that has a
// controller-runtime manager wraps the controller with kube.Runnable and
// hands the result to mgr.Add; the wrapper is where leader election is
// decided. A process without a manager starts the controller itself.
//
// A controller that runs in every replica must have an idempotent
// Reconcile, since every replica also resyncs, and its event sources should
// share events across replicas rather than deliver every event everywhere;
// see nats.Source.QueueGroup.
//
// # Dependencies
//
// The work queue is client-go's workqueue, used for its deduplication,
// in-flight tracking, delayed adds and rate limiting. It is internal to the
// Controller; sources only ever see a channel.
package reconciler
