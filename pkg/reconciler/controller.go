package reconciler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"k8s.io/client-go/util/workqueue"

	"github.com/akuity/kargo/pkg/logging"
)

// requestBuffer is how many pushed requests may wait for the controller to
// move them into the queue. The move is immediate, so this only absorbs
// bursts; a source blocks briefly if it fills.
const requestBuffer = 128

// Controller reconciles the requests its sources push. Build one with New
// and call Start, or run it under a controller-runtime manager through
// package kube.
type Controller[request comparable] struct {
	// Name identifies the controller in logs and names its queue. Required;
	// no whitespace.
	Name string
	// Reconciler reconciles each request. Required.
	Reconciler Reconciler[request]
	// Sources push the requests to reconcile. At least one is required. They
	// are started one at a time, in this order; list event-driven sources
	// before listing ones.
	Sources []Source[request]
	// Workers is how many requests are reconciled concurrently. Zero means
	// one. No request is ever reconciled by two workers at once.
	Workers int
}

// Start starts every source, in order, then runs the workers until ctx is
// done. It returns an error if the controller is misconfigured or a source
// cannot start; failures to reconcile are retried and logged.
//
// A source's Start returns once its initial state has been pushed, so the
// workers, which start after the last source, begin with the complete boot
// relist in the queue.
func (c *Controller[request]) Start(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}
	logger := logging.LoggerFromContext(ctx).WithValues("controller", c.Name)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Failed requests are retried with client-go's default rate limiter:
	// per-request exponential backoff from 5ms, capped overall.
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[request](),
		workqueue.TypedRateLimitingQueueConfig[request]{Name: c.Name},
	)
	defer queue.ShutDown()

	// Sources stop pushing when this context is done, including when a later
	// source fails to start.
	sourceCtx, stopSources := context.WithCancel(ctx)
	defer stopSources()

	// Move pushed requests into the queue. This runs before any source starts
	// so that a source loading its initial state is never blocked for long.
	requests := make(chan request, requestBuffer)
	go func() {
		for {
			select {
			case <-sourceCtx.Done():
				return
			case req := <-requests:
				queue.Add(req)
			}
		}
	}()

	for i, src := range c.Sources {
		if err := src.Start(sourceCtx, requests); err != nil {
			return fmt.Errorf("error starting source %d of controller %q: %w", i, c.Name, err)
		}
	}

	workers := max(c.Workers, 1)
	logger.Info("starting controller", "workers", workers, "sources", len(c.Sources))
	var wg sync.WaitGroup
	for range workers {
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

func (c *Controller[request]) validate() error {
	if c.Name == "" {
		return errors.New("controller name is required")
	}
	if strings.ContainsFunc(c.Name, unicode.IsSpace) {
		return fmt.Errorf("controller name %q must not contain whitespace", c.Name)
	}
	if c.Reconciler == nil {
		return fmt.Errorf("controller %q requires a reconciler", c.Name)
	}
	if len(c.Sources) == 0 {
		return fmt.Errorf("controller %q watches nothing", c.Name)
	}
	return nil
}

// processNext reconciles the next request in the queue. It returns false
// once the queue has been shut down.
//
// TODO: record reconcile duration and outcome, and the queue's depth and
// latency, through pkg/metrics.
func (c *Controller[request]) processNext(
	ctx context.Context,
	queue workqueue.TypedRateLimitingInterface[request],
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
			logging.LoggerFromContext(ctx).Error(err, "error reconciling", "request", req)
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
// bad request cannot take down the process.
func (c *Controller[request]) reconcile(ctx context.Context, req request) (result Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic reconciling %v: %v", req, r)
		}
	}()
	return c.Reconciler.Reconcile(ctx, req)
}
