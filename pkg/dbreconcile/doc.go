// Package dbreconcile provides controllers for resources that live in the
// database rather than in Kubernetes.
//
// It borrows controller-runtime's model and vocabulary. A controller owns one
// kind of resource, identified by a key of type K. Events about resources
// arrive from sources, are filtered by predicates, are turned into keys by
// event handlers, and land in a work queue. Workers take keys from the queue
// and hand each to a Reconciler as a Request:
//
//	Source -> Predicates -> EventHandler -> work queue -> Reconciler
//
// The queue deduplicates: however many events arrive for a key while it waits
// or is being reconciled, it is reconciled once more, not once per event. A
// Reconcile that fails is retried with per-key exponential backoff, and one
// that returns a Result with RequeueAfter is reconciled again after that long.
//
// A Request carries only the key. Reconcile reads the resource fresh from the
// database and acts on what it finds, so it never acts on a stale copy and
// never needs to know which event caused it to run. That is what makes a
// controller level-triggered: an event says "look at this again", not "this
// is what changed".
//
// # Sources and events
//
// Events travel over NATS. Each resource has its own subject tree, named
// kargo.<resource>.<event>, such as kargo.promotionrequests.created. A
// component that writes a resource publishes an Event with Publish once its
// write has committed, and a controller subscribes with Subject, usually with
// a wildcard such as kargo.promotionrequests.>. An Event carries its kind,
// the resource's key and the resource before and after the change, as JSON.
//
// The resource travels in the database's own representation: the type the
// store reads and writes, not an API type. That is what the publisher has just
// written and what the reconciler will read, and it keeps events independent
// of how the resource happens to be presented to users.
//
// Core NATS delivers a message only to subscribers connected when it is
// published, so events can be lost: while a controller restarts, say. A
// controller should therefore also resync. Resync enumerates, with a Lister,
// every key that needs work and enqueues it, on a fixed interval. Resynced
// keys bypass predicates, which only have events to judge, so a lost event
// never leaves a resource unreconciled for longer than the interval.
//
// # Watching other resources
//
// A controller may watch any number of subjects. Events about its own
// resource usually map to their own key with EnqueueKey. Events about another
// resource map to the keys of the resources they affect with EnqueueMapped,
// as controller-runtime's EnqueueRequestsFromMapFunc does. Everything ends up
// in the controller's single queue, so a resource that several events affect
// at once is still reconciled once.
//
// # Replicas
//
// Controllers do not take part in leader election, so every replica runs
// every controller. The replicas share events rather than each receiving them
// all: each of a controller's watches subscribes in a NATS queue group named
// after the controller and the watch, and NATS delivers each message to one
// member of a group. Other controllers watching the same subject are in
// groups of their own and still receive every event. A watch whose replicas
// each need every event opts out with WithoutQueueGroup.
//
// Because the group is derived from the controller's name, replicas that must
// not share events need distinct names; a sharded controller includes its
// shard's name in its own.
//
// Sharing events does not make reconciles exclusive. Two events for one key
// may reach different replicas, and every replica resyncs, so two replicas
// may reconcile the same key at the same time. Reconcilers must therefore be
// idempotent: the second reconcile must be harmless.
//
// # Example
//
//	type event = dbreconcile.Event[uuid.UUID, database.PromotionRequestSnapshot]
//
//	err := dbreconcile.NewControllerManagedBy[uuid.UUID](mgr).
//		Named("promotion-requests").
//		Watches(dbreconcile.NewWatch(
//			dbreconcile.Subject[uuid.UUID, database.PromotionRequestSnapshot](conn, "kargo.promotionrequests.>"),
//			dbreconcile.EnqueueKey[uuid.UUID, database.PromotionRequestSnapshot](),
//			dbreconcile.Predicate[uuid.UUID, database.PromotionRequestSnapshot]{
//				// Only a change to the request's Targets warrants a reconcile.
//				Update: func(e event) bool {
//					return !slices.Equal(targetNames(e.Old), targetNames(e.New))
//				},
//			},
//		)).
//		Resync(dbreconcile.ListerFunc[uuid.UUID](store.ListOpenPromotionRequestIDs), 2*time.Minute).
//		WithWorkers(4).
//		Complete(reconciler)
package dbreconcile
