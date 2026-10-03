// Package kube is the Kubernetes implementation of a reconciler.Source: a
// controller-runtime cache watched for changes to objects of one kind. It is
// the only part of the framework that knows about Kubernetes objects, and
// the only one that imports controller-runtime's source and handler
// packages. It does so to reuse the informer plumbing and so that the
// handlers and predicates existing controllers already have carry over
// unchanged.
package kube

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/akuity/kargo/pkg/reconciler"
)

// kind is a reconciler.Source over controller-runtime's Kind source. The
// inner source starts without blocking, reports the result of its initial
// list through WaitForSync, and pushes into a work queue; this one waits for
// the list before returning and gives the inner source a queue that forwards
// to the controller's channel.
type kind[request comparable] struct {
	inner source.TypedSyncingSource[request]
}

// Kind returns a source that lists every object of the given kind when it
// starts and pushes whatever controller-runtime's handler maps each object
// to, then keeps pushing as objects change, filtered by the predicates. The
// handler determines the request type, so it serves controllers keyed by
// reconcile.Request and by anything else alike.
//
// Start fails when the kind is not installed, not in the scheme, or cannot
// be listed, and the controller does not run. The cache must be started by
// whoever owns it, usually a manager; the manager starts its cache before
// any controller.
func Kind[object client.Object, request comparable](
	c cache.Cache,
	obj object,
	h handler.TypedEventHandler[object, request],
	predicates ...predicate.TypedPredicate[object],
) reconciler.Source[request] {
	return &kind[request]{inner: source.TypedKind(c, obj, h, predicates...)}
}

// Keyed returns a Kind source for a controller keyed by K, mapping each
// changed object to the keys it affects: a Promotion to the Targets it
// promoted to, say. A controller for Targets uses it to watch Promotions.
func Keyed[object client.Object, K comparable](
	c cache.Cache,
	obj object,
	keys func(context.Context, object) []K,
	predicates ...predicate.TypedPredicate[object],
) reconciler.Source[K] {
	return Kind(c, obj, handler.TypedEnqueueRequestsFromMapFunc(keys), predicates...)
}

// Start implements reconciler.Source. It starts the informer-backed source
// and blocks until its initial list has been pushed.
func (k *kind[request]) Start(ctx context.Context, out chan<- request) error {
	if err := k.inner.Start(ctx, &channelQueue[request]{ctx: ctx, out: out}); err != nil {
		return err
	}
	if err := k.inner.WaitForSync(ctx); err != nil {
		return fmt.Errorf("error listing kind: %w", err)
	}
	return nil
}

// channelQueue is the least work queue controller-runtime's source and
// handlers will accept. Handlers only ever Add; Add forwards to the
// controller's channel. The rest of the interface exists for a controller
// that owns the queue, which this is not, and is inert.
type channelQueue[request comparable] struct {
	ctx context.Context
	out chan<- request
}

var _ workqueue.TypedRateLimitingInterface[int] = (*channelQueue[int])(nil)

func (q *channelQueue[request]) Add(item request) {
	select {
	case q.out <- item:
	case <-q.ctx.Done():
	}
}

// AddAfter is honored, although no handler of ours uses it.
func (q *channelQueue[request]) AddAfter(item request, d time.Duration) {
	time.AfterFunc(d, func() { q.Add(item) })
}

// AddRateLimited has no rate limiter to consult here; the controller's queue
// applies its own when the request fails.
func (q *channelQueue[request]) AddRateLimited(item request) { q.Add(item) }
func (q *channelQueue[request]) Forget(request)              {}
func (q *channelQueue[request]) NumRequeues(request) int     { return 0 }
func (q *channelQueue[request]) Len() int                    { return 0 }
func (q *channelQueue[request]) Done(request)                {}
func (q *channelQueue[request]) ShutDown()                   {}
func (q *channelQueue[request]) ShutDownWithDrain()          {}
func (q *channelQueue[request]) ShuttingDown() bool          { return q.ctx.Err() != nil }

func (q *channelQueue[request]) Get() (request, bool) {
	var zero request
	return zero, true
}
