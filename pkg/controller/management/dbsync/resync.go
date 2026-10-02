package dbsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/logging"
)

type cacheSyncer interface {
	WaitForCacheSync(context.Context) bool
}

type resyncRunner struct {
	cache    cacheSyncer
	targets  []resyncTarget
	interval time.Duration
}

type resyncTarget struct {
	name   string
	syncer syncapi.Syncer
	events chan<- event.GenericEvent
}

func (r *resyncRunner) Start(ctx context.Context) error {
	if !r.cache.WaitForCacheSync(ctx) {
		return nil
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		if err := r.resync(ctx); err != nil && ctx.Err() == nil {
			// Resync failure must not terminate the manager's unrelated controllers.
			logging.LoggerFromContext(ctx).Error(err, "error resyncing database mirror")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *resyncRunner) resync(ctx context.Context) error {
	var errs []error
	for _, target := range r.targets {
		if err := target.resync(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error resyncing %s: %w", target.name, err))
		}
	}
	return errors.Join(errs...)
}

func (t resyncTarget) resync(ctx context.Context) error {
	changes, err := t.syncer.Diff(ctx)
	if err != nil {
		return err
	}
	for _, key := range changes.ToSync {
		// Only the key is delivered to the queue. Reconciliation reads a fresh
		// object so it never writes the snapshot used for comparison.
		obj := t.syncer.NewObject()
		obj.SetNamespace(key.Namespace)
		obj.SetName(key.Name)
		select {
		case t.events <- event.GenericEvent{Object: obj}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if len(changes.ToDelete) > 0 {
		return t.syncer.DeleteByIDs(ctx, changes.ToDelete)
	}
	return nil
}
