// Package syncapi defines the contract for mirroring a Kubernetes resource kind.
package syncapi

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/akuity/kargo/pkg/reconciler"
)

// Syncer owns the resource-specific mapping and storage operations. The shared
// controller supplies current objects from an uncached Kubernetes reader.
type Syncer interface {
	NewObject() client.Object
	// Sources returns the event sources that push keys for this kind: the
	// kind itself, usually through kube.Objects, and any other kind whose
	// changes must re-sync rows, such as a parent whose identity a row
	// records. The controller adds the periodic Diff after them.
	Sources(cache.Cache) []reconciler.Source[reconcile.Request]
	Sync(context.Context, client.Object) error
	// Delete is called only after Kubernetes confirms that the key is absent.
	Delete(context.Context, client.ObjectKey) error
	// Diff snapshots database rows before obtaining a complete Kubernetes list
	// and returns every key whose mirror needs attention: objects with a
	// missing or differing row, and rows whose object is gone. It performs no
	// writes and returns nothing on a failed or incomplete list. The list may
	// come from the cache: reconciling a returned key reads Kubernetes afresh
	// before upserting or deleting, so a stale entry costs at most one extra
	// reconcile or one more interval, never a wrong write.
	Diff(context.Context) ([]reconcile.Request, error)
}
