package dbreconcile

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/akuity/kargo/pkg/logging"
)

const (
	defaultWorkers        = 1
	defaultResyncPageSize = 100
)

// Request names the resource to reconcile, as a controller-runtime Request
// names an object. It carries only the key; the Reconciler reads the resource
// itself.
type Request[K comparable] struct {
	Key K
}

// Result says what should happen after a successful Reconcile, as a
// controller-runtime Result does.
type Result struct {
	// RequeueAfter, when positive, reconciles the same key again after that
	// long.
	RequeueAfter time.Duration
}

// Reconciler reconciles one resource. It is handed only the resource's key,
// reads the resource fresh and acts on what it finds. Returning an error
// retries the key with exponential backoff. A Reconciler must be idempotent;
// see the package documentation.
type Reconciler[K comparable] interface {
	Reconcile(ctx context.Context, req Request[K]) (Result, error)
}

// ReconcilerFunc is a Reconciler implemented by a function.
type ReconcilerFunc[K comparable] func(context.Context, Request[K]) (Result, error)

// Reconcile implements Reconciler.
func (f ReconcilerFunc[K]) Reconcile(ctx context.Context, req Request[K]) (Result, error) {
	return f(ctx, req)
}

// Watch is a source of events together with the predicates that filter them
// and the handler that turns them into a controller's keys. Create one with
// NewWatch.
type Watch[K comparable] interface {
	start(ctx context.Context, enqueue func(K), queueGroup string) error
}

type watch[SK comparable, T any, K comparable] struct {
	source     Source[SK, T]
	handler    EventHandler[SK, T, K]
	predicates []Predicate[SK, T]
}

// NewWatch combines a source, the handler that turns its events into a
// controller's keys, and any predicates that events must satisfy on the way.
// An event must be let through by every predicate.
//
// The three are combined by a function, rather than passed to the builder's
// Watches separately as they are in controller-runtime, because the source's
// key and resource types need not be the controller's, and Go methods cannot
// introduce type parameters of their own.
func NewWatch[SK comparable, T any, K comparable](
	source Source[SK, T],
	handler EventHandler[SK, T, K],
	predicates ...Predicate[SK, T],
) Watch[K] {
	return &watch[SK, T, K]{
		source:     source,
		handler:    handler,
		predicates: predicates,
	}
}

func (w *watch[SK, T, K]) start(ctx context.Context, enqueue func(K), queueGroup string) error {
	if w.source == nil {
		return errors.New("a watch requires a source")
	}
	if w.handler == nil {
		return errors.New("a watch requires an event handler")
	}
	return w.source.Start(ctx, func(ctx context.Context, e Event[SK, T]) {
		for _, predicate := range w.predicates {
			if !predicate.Allow(e) {
				return
			}
		}
		for _, key := range w.handler.Keys(ctx, e) {
			enqueue(key)
		}
	}, queueGroup)
}

// Manager is what a controller is added to: a controller-runtime Manager,
// usually.
type Manager interface {
	Add(manager.Runnable) error
}

// Builder builds a controller, as controller-runtime's builder does. Start one
// with NewControllerManagedBy.
type Builder[K comparable] struct {
	mgr            Manager
	name           string
	watches        []Watch[K]
	lister         Lister[K]
	resyncInterval time.Duration
	resyncPageSize int
	workers        int
	rateLimiter    workqueue.TypedRateLimiter[Request[K]]
}

// NewControllerManagedBy starts building a controller for resources
// identified by keys of type K, to be added to the given manager.
func NewControllerManagedBy[K comparable](mgr Manager) *Builder[K] {
	return &Builder[K]{
		mgr:            mgr,
		resyncPageSize: defaultResyncPageSize,
		workers:        defaultWorkers,
	}
}

// Named sets the controller's name, which identifies it in logs and metrics
// and names the queue groups its replicas share events through. Controllers
// whose replicas must not share events need distinct names: a sharded
// controller includes its shard's name in its own.
func (b *Builder[K]) Named(name string) *Builder[K] {
	b.name = name
	return b
}

// Watches adds a source of events for the controller. A controller may watch
// any number of sources.
func (b *Builder[K]) Watches(w Watch[K]) *Builder[K] {
	b.watches = append(b.watches, w)
	return b
}

// Resync enqueues every key the lister enumerates, once when the controller
// starts and then on every tick of interval. Resynced keys bypass predicates.
func (b *Builder[K]) Resync(lister Lister[K], interval time.Duration) *Builder[K] {
	b.lister = lister
	b.resyncInterval = interval
	return b
}

// WithResyncPageSize sets how many keys the resync requests from its lister
// at a time. It defaults to 100.
func (b *Builder[K]) WithResyncPageSize(size int) *Builder[K] {
	b.resyncPageSize = size
	return b
}

// WithWorkers sets how many keys are reconciled concurrently. It defaults to
// one. No key is ever reconciled by two workers at once.
func (b *Builder[K]) WithWorkers(workers int) *Builder[K] {
	b.workers = workers
	return b
}

// WithRateLimiter sets how failed keys are retried. It defaults to
// controller-runtime's default: per-key exponential backoff, capped overall.
func (b *Builder[K]) WithRateLimiter(rateLimiter workqueue.TypedRateLimiter[Request[K]]) *Builder[K] {
	b.rateLimiter = rateLimiter
	return b
}

// Build returns the controller without adding it to the manager.
func (b *Builder[K]) Build(reconciler Reconciler[K]) (*Controller[K], error) {
	if b.name == "" {
		return nil, errors.New("controller name is required")
	}
	// The name becomes part of queue group names, which may not contain
	// whitespace.
	if strings.ContainsFunc(b.name, unicode.IsSpace) {
		return nil, fmt.Errorf("controller name %q must not contain whitespace", b.name)
	}
	if reconciler == nil {
		return nil, fmt.Errorf("controller %q requires a reconciler", b.name)
	}
	if len(b.watches) == 0 && b.lister == nil {
		return nil, fmt.Errorf("controller %q watches nothing and never resyncs", b.name)
	}
	if b.lister != nil && b.resyncInterval <= 0 {
		return nil, fmt.Errorf("controller %q requires a positive resync interval", b.name)
	}
	if b.resyncPageSize < 1 {
		return nil, fmt.Errorf("controller %q requires a positive resync page size", b.name)
	}
	if b.workers < 1 {
		return nil, fmt.Errorf("controller %q requires at least one worker", b.name)
	}
	rateLimiter := b.rateLimiter
	if rateLimiter == nil {
		rateLimiter = workqueue.DefaultTypedControllerRateLimiter[Request[K]]()
	}
	return &Controller[K]{
		name:           b.name,
		reconciler:     reconciler,
		watches:        b.watches,
		lister:         b.lister,
		resyncInterval: b.resyncInterval,
		resyncPageSize: b.resyncPageSize,
		workers:        b.workers,
		rateLimiter:    rateLimiter,
	}, nil
}

// Complete builds the controller and adds it to the manager, which starts it.
func (b *Builder[K]) Complete(reconciler Reconciler[K]) error {
	if b.mgr == nil {
		return errors.New("a manager is required")
	}
	c, err := b.Build(reconciler)
	if err != nil {
		return err
	}
	if err = b.mgr.Add(c); err != nil {
		return fmt.Errorf("error adding controller %q to manager: %w", b.name, err)
	}
	return nil
}

// Controller reconciles resources identified by keys of type K. It is a
// controller-runtime Runnable; build one with a Builder.
type Controller[K comparable] struct {
	name           string
	reconciler     Reconciler[K]
	watches        []Watch[K]
	lister         Lister[K]
	resyncInterval time.Duration
	resyncPageSize int
	workers        int
	rateLimiter    workqueue.TypedRateLimiter[Request[K]]
}

// Start implements manager.Runnable. It starts every watch, then the resync
// and the workers, and runs until ctx is done. It returns an error only when
// a watch cannot be started; failures to reconcile are retried and logged.
func (c *Controller[K]) Start(ctx context.Context) error {
	logger := logging.LoggerFromContext(ctx).WithValues("controller", c.name)
	ctx = logging.ContextWithLogger(ctx, logger)

	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		c.rateLimiter,
		workqueue.TypedRateLimitingQueueConfig[Request[K]]{Name: c.name},
	)
	enqueue := keyQueue[K]{queue}.add

	// Watches stop delivering when this context is done, including when a
	// later watch fails to start.
	watchCtx, stopWatches := context.WithCancel(ctx)
	defer stopWatches()
	for i, w := range c.watches {
		if err := w.start(watchCtx, enqueue, queueGroup(c.name, i)); err != nil {
			queue.ShutDown()
			return fmt.Errorf("error starting watch %d of controller %q: %w", i, c.name, err)
		}
	}

	logger.Info(
		"starting controller",
		"workers", c.workers,
		"watches", len(c.watches),
		"resyncInterval", c.resyncInterval,
	)

	var wg sync.WaitGroup
	if c.lister != nil {
		wg.Go(func() { c.runResync(ctx, enqueue) })
	}
	for range c.workers {
		wg.Go(func() {
			for c.processNext(ctx, queue) {
			}
		})
	}

	<-ctx.Done()
	logger.Info("stopping controller")
	queue.ShutDown()
	wg.Wait()
	return nil
}

// keyQueue adapts the work queue to the func(K) that watches and the resync
// enqueue keys with.
//
// It is a method rather than a closure over the queue: golangci-lint's unparam
// linter crashes on a closure, within a generic function, whose parameter is
// a bare type parameter.
type keyQueue[K comparable] struct {
	queue workqueue.TypedRateLimitingInterface[Request[K]]
}

func (q keyQueue[K]) add(key K) {
	q.queue.Add(Request[K]{Key: key})
}

// queueGroup names the queue group through which the replicas of a
// controller share the events of one of its watches. The group is per watch,
// not per controller: subscriptions in one group split its messages even
// within one process, so two watches of one controller sharing a group would
// each see only some of the events on a subject they both watch.
func queueGroup(controller string, watch int) string {
	return controller + "." + strconv.Itoa(watch)
}

// NeedLeaderElection implements manager.LeaderElectionRunnable. Controllers
// run in every replica, so Reconcilers must be idempotent.
func (c *Controller[K]) NeedLeaderElection() bool { return false }

// processNext reconciles the next key in the queue. It returns false once the
// queue has been shut down.
func (c *Controller[K]) processNext(
	ctx context.Context,
	queue workqueue.TypedRateLimitingInterface[Request[K]],
) bool {
	req, shutdown := queue.Get()
	if shutdown {
		return false
	}
	defer queue.Done(req)

	result, err := c.reconcile(ctx, req)
	switch {
	case err != nil:
		if ctx.Err() == nil {
			logging.LoggerFromContext(ctx).Error(err, "error reconciling", "key", req.Key)
		}
		queue.AddRateLimited(req)
	case result.RequeueAfter > 0:
		queue.Forget(req)
		queue.AddAfter(req, result.RequeueAfter)
	default:
		queue.Forget(req)
	}
	return true
}

// reconcile calls the Reconciler, turning a panic into an error so that one
// bad key cannot take down the process.
func (c *Controller[K]) reconcile(ctx context.Context, req Request[K]) (result Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic reconciling key %v: %v", req.Key, r)
		}
	}()
	return c.reconciler.Reconcile(ctx, req)
}

// runResync resyncs immediately and then on every tick until ctx is done.
func (c *Controller[K]) runResync(ctx context.Context, enqueue func(K)) {
	logger := logging.LoggerFromContext(ctx)
	ticker := time.NewTicker(c.resyncInterval)
	defer ticker.Stop()
	for {
		if err := c.resync(ctx, enqueue); err != nil && ctx.Err() == nil {
			logger.Error(err, "error resyncing")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// resync pages through every key the lister enumerates and enqueues it.
func (c *Controller[K]) resync(ctx context.Context, enqueue func(K)) error {
	var after K
	for {
		keys, err := c.lister.ListKeys(ctx, after, c.resyncPageSize)
		if err != nil {
			return fmt.Errorf("error listing keys: %w", err)
		}
		for _, key := range keys {
			enqueue(key)
		}
		if len(keys) < c.resyncPageSize {
			return nil
		}
		last := keys[len(keys)-1]
		// A full page that does not advance would be requested forever.
		if last == after {
			return fmt.Errorf("lister returned a full page without advancing past key %v", after)
		}
		after = last
	}
}
