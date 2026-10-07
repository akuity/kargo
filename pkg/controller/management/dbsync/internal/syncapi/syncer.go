// Package syncapi defines the contract for mirroring a Kubernetes resource kind.
package syncapi

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Syncer owns the resource-specific mapping and storage operations. The shared
// controller supplies current objects from an uncached Kubernetes reader.
type Syncer interface {
	NewObject() client.Object
	Sync(context.Context, client.Object) error
	// Delete is called only after Kubernetes confirms that the key is absent.
	Delete(context.Context, client.ObjectKey) error
	// Diff snapshots database rows before obtaining a complete Kubernetes list
	// and returns every key whose mirror needs attention: objects with a
	// missing or differing row, and rows whose object is gone. It performs no
	// writes and returns nothing on a failed or incomplete list. Reconciling a
	// returned key reads Kubernetes again and upserts or deletes accordingly.
	Diff(context.Context) ([]reconcile.Request, error)
}
