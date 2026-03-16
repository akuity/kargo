package kubeclient

import (
	"context"
	"strconv"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VersionWaiter is the interface used by reconcilers to wait for the informer
// cache to observe a specific resourceVersion after a status patch.
type VersionWaiter interface {
	WaitForVersion(ctx context.Context, key types.NamespacedName, rv string, timeout time.Duration) error
}

// VersionFence tracks resourceVersions observed by an informer cache and
// allows callers to block until a specific version has been observed.
// This prevents acting on stale cached data after a status patch.
//
// Attach it to an informer via cache.GetInformer() + AddEventHandler().
type VersionFence struct {
	mu       sync.Mutex
	cond     *sync.Cond
	observed map[types.NamespacedName]int64
}

// NewVersionFence creates a new VersionFence.
func NewVersionFence() *VersionFence {
	f := &VersionFence{
		observed: make(map[types.NamespacedName]int64),
	}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// OnAdd implements cache.ResourceEventHandler.
func (f *VersionFence) OnAdd(obj any, _ bool) {
	f.recordVersion(obj)
}

// OnUpdate implements cache.ResourceEventHandler.
func (f *VersionFence) OnUpdate(_, newObj any) {
	f.recordVersion(newObj)
}

// OnDelete implements cache.ResourceEventHandler.
func (f *VersionFence) OnDelete(obj any) {
	o, ok := obj.(client.Object)
	if !ok {
		return
	}
	key := types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.observed, key)
}

func (f *VersionFence) recordVersion(obj any) {
	o, ok := obj.(client.Object)
	if !ok {
		return
	}
	rv, err := strconv.ParseInt(o.GetResourceVersion(), 10, 64)
	if err != nil {
		return
	}
	key := types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()}
	f.mu.Lock()
	defer f.mu.Unlock()
	if rv > f.observed[key] {
		f.observed[key] = rv
	}
	f.cond.Broadcast()
}

// WaitForVersion blocks until the cache has observed at least the given
// resourceVersion for the specified object, or until the timeout elapses
// or ctx is canceled. Returns nil if the version was observed, or the
// context error on timeout/cancellation.
func (f *VersionFence) WaitForVersion(
	ctx context.Context,
	key types.NamespacedName,
	rv string,
	timeout time.Duration,
) error {
	target, err := strconv.ParseInt(rv, 10, 64)
	if err != nil {
		return nil // Non-integer RV, skip waiting
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Unblock cond.Wait() when the context is done.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			f.cond.Broadcast()
		case <-done:
		}
	}()

	f.mu.Lock()
	defer f.mu.Unlock()
	for f.observed[key] < target {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		f.cond.Wait()
	}
	return nil
}
