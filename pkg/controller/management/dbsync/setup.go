// Package dbsync mirrors Kubernetes identities into PostgreSQL.
package dbsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
	numWorkers     = 4
)

// SetupWithManager registers one controller per mirrored kind with the
// manager. The caller owns the database pool and its lifetime.
func SetupWithManager(ctx context.Context, mgr manager.Manager, store database.Store) error {
	reader := mgr.GetAPIReader()
	if err := register(mgr, reader, &kargoapi.Project{}, projects.NewSyncer(reader, store)); err != nil {
		return err
	}
	logging.LoggerFromContext(ctx).Info("Initialized database synchronization")
	return nil
}

// register builds the controller that mirrors one kind and adds it to the
// manager. The controller watches the kind through the manager's cache and
// runs the syncer's Diff on an interval; both push keys that the reconciler
// resolves by reading Kubernetes afresh.
func register[O client.Object](
	mgr manager.Manager,
	reader client.Reader,
	obj O,
	syncer syncapi.Syncer,
) error {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("error identifying database sync resource: %w", err)
	}
	name := "db-sync-" + strings.ToLower(gvk.Kind)
	c, err := reconciler.New[reconcile.Request](name).
		// Every object of the kind at startup, then every change. The
		// predicate drops the informer's periodic resync events, whose
		// objects have not changed.
		Watch(kube.Kind(
			mgr.GetCache(),
			obj,
			&handler.TypedEnqueueRequestForObject[O]{},
			predicate.TypedResourceVersionChangedPredicate[O]{},
		)).
		// Whatever the database disagrees with, at startup and on an
		// interval. Listed last so the watch above is already subscribed
		// while the diff runs.
		Watch(list.New(syncer.Diff).Every(resyncInterval)).
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
