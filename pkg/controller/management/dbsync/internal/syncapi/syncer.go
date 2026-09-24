// Package syncapi defines the contract for mirroring a Kubernetes resource kind.
package syncapi

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Syncer owns the resource-specific mapping and storage operations. The shared
// controller supplies current objects from an uncached Kubernetes reader.
type Syncer interface {
	NewObject() client.Object
	Sync(context.Context, client.Object) error
	// Delete is called only after Kubernetes confirms that the key is absent.
	Delete(context.Context, client.ObjectKey) error
	// Diff snapshots database rows before obtaining complete Kubernetes lists.
	// It performs no writes and returns no changes on a failed or incomplete list.
	Diff(context.Context) (Changes, error)
	DeleteByIDs(context.Context, []string) error
}

// Changes identifies work needed to bring one resource's mirror up to date.
type Changes struct {
	ToSync   []client.ObjectKey
	ToDelete []string // Only UIDs from the database snapshot absent from Kubernetes.
}
