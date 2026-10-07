// Package dbsync mirrors Kubernetes identities into PostgreSQL.
package dbsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/controller/management/dbsync/projects"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/reconciler/kube"
	"github.com/akuity/kargo/pkg/reconciler/list"
)

const (
	resyncInterval = time.Minute
	// resyncJitter spreads the diffs of the mirrored kinds so they do not
	// all list the database and Kubernetes at the same instant.
	resyncJitter = 0.1
	numWorkers   = 4
)

// SetupWithManager registers one controller per mirrored kind with the
// manager. The caller owns the database pool and its lifetime.
func SetupWithManager(ctx context.Context, mgr manager.Manager, store database.Store) error {
	reader := mgr.GetAPIReader()
	// Add a kind by appending its syncer here.
	syncers := []syncapi.Syncer{
		projects.NewSyncer(reader, store),
	}
	for _, syncer := range syncers {
		if err := register(mgr, reader, syncer); err != nil {
			return err
		}
	}
	logging.LoggerFromContext(ctx).Info("Initialized database synchronization")
	return nil
}

// register builds the controller that mirrors the syncer's kind and adds it
// to the manager. The controller watches the syncer's sources through the
// manager's cache and runs the syncer's Diff on an interval; all of them push
// keys that the reconciler resolves by reading Kubernetes afresh.
func register(mgr manager.Manager, reader client.Reader, syncer syncapi.Syncer) error {
	gvk, err := apiutil.GVKForObject(syncer.NewObject(), mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("error identifying database sync resource: %w", err)
	}
	name := "db-sync-" + strings.ToLower(gvk.Kind)
	c, err := reconciler.New[reconcile.Request](name).
		Watch(syncer.Sources(mgr.GetCache())...).
		// Whatever the database disagrees with, at startup and on an
		// interval. Listed last so the watches above are already subscribed
		// while the diff runs.
		Watch(list.New(syncer.Diff).Every(resyncInterval).Jitter(resyncJitter)).
		Workers(numWorkers).
		Func(newReconciler(reader, syncer))
	if err != nil {
		return fmt.Errorf("error building %s: %w", name, err)
	}
	if err = mgr.Add(kube.Runnable(c)); err != nil {
		return fmt.Errorf("error adding %s to manager: %w", name, err)
	}
	return nil
}
