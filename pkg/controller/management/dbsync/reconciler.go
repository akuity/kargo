package dbsync

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/akuity/kargo/pkg/controller/management/dbsync/internal/syncapi"
	"github.com/akuity/kargo/pkg/reconciler"
)

// newReconciler returns the reconcile function shared by every mirrored kind.
// It is handed only a key, whichever source pushed it, reads the object fresh
// and upserts its row, or deletes the row once Kubernetes confirms the object
// is gone.
func newReconciler(reader client.Reader, syncer syncapi.Syncer) reconciler.Func[reconcile.Request] {
	return func(ctx context.Context, req reconcile.Request) (reconciler.Result, error) {
		obj := syncer.NewObject()
		if err := reader.Get(ctx, req.NamespacedName, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				return reconciler.Result{}, fmt.Errorf("error reading object to sync: %w", err)
			}
			return reconciler.Result{}, syncer.Delete(ctx, req.NamespacedName)
		}
		return reconciler.Result{}, syncer.Sync(ctx, obj)
	}
}
