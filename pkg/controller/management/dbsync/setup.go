// Package dbsync mirrors Kubernetes identities into PostgreSQL.
package dbsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/projects"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/stages"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	resyncInterval = time.Minute
	eventBuffer    = 128
	numWorkers     = 4
)

// SetupWithManager registers independent Project and Stage controllers and
// periodic resync. The caller owns the database pool and its lifetime.
func SetupWithManager(ctx context.Context, mgr manager.Manager, store database.Store) error {
	reader := mgr.GetAPIReader()
	syncers := []syncapi.Syncer{
		projects.NewSyncer(reader, store),
		stages.NewSyncer(reader, store),
	}
	targets := make([]resyncTarget, 0, len(syncers))
	for _, syncer := range syncers {
		obj := syncer.NewObject()
		gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
		if err != nil {
			return fmt.Errorf("error identifying database sync resource: %w", err)
		}
		name := "db-sync-" + strings.ToLower(gvk.Kind)
		events := make(chan event.GenericEvent, eventBuffer)
		err = ctrl.NewControllerManagedBy(mgr).
			Named(name).
			For(obj).
			WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
			WithOptions(controller.CommonOptions(numWorkers)).
			WatchesRawSource(source.Channel(events, &handler.EnqueueRequestForObject{})).
			Complete(newReconciler(reader, syncer))
		if err != nil {
			return fmt.Errorf("error setting up %s: %w", name, err)
		}
		targets = append(targets, resyncTarget{name: name, syncer: syncer, events: events})
	}
	if err := mgr.Add(&resyncRunner{
		cache:    mgr.GetCache(),
		targets:  targets,
		interval: resyncInterval,
	}); err != nil {
		return fmt.Errorf("error setting up database resync: %w", err)
	}
	logging.LoggerFromContext(ctx).Info("Initialized database synchronization")
	return nil
}
