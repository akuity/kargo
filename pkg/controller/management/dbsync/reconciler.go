package dbsync

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
)

type reconciler struct {
	reader client.Reader
	syncer syncapi.Syncer
}

func newReconciler(reader client.Reader, syncer syncapi.Syncer) reconcile.Reconciler {
	return &reconciler{reader: reader, syncer: syncer}
}

func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	obj := r.syncer.NewObject()
	if err := r.reader.Get(ctx, req.NamespacedName, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, fmt.Errorf("error reading object to sync: %w", err)
		}
		return reconcile.Result{}, r.syncer.Delete(ctx, req.NamespacedName)
	}
	return reconcile.Result{}, r.syncer.Sync(ctx, obj)
}
