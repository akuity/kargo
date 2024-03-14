package v1alpha1

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetWarehouse returns a pointer to the Warehouse resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetWarehouse(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Warehouse, error) {
	warehouse := Warehouse{}
	if err := c.Get(ctx, namespacedName, &warehouse); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Warehouse %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &warehouse, nil
}

// RefreshWarehouse forces reconciliation of a Warehouse by setting an annotation
// on the Warehouse, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshWarehouse(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Warehouse, error) {
	warehouse := &Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(
		ctx,
		c,
		warehouse,
		AnnotationKeyRefresh,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return warehouse, nil
}

// ClearWarehouseRefresh is called by the Warehouse controller to clear the refresh
// annotation on the Warehouse (if present). A client (e.g. UI) who requested a
// Warehouse refresh, can wait until the annotation is cleared, to understand that
// the controller successfully reconciled the Warehouse after the refresh request.
func ClearWarehouseRefresh(
	ctx context.Context,
	c client.Client,
	wh *Warehouse,
) error {
	if wh.Annotations == nil {
		return nil
	}
	if _, ok := wh.Annotations[AnnotationKeyRefresh]; !ok {
		return nil
	}
	newWh := Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wh.Name,
			Namespace: wh.Namespace,
		},
	}
	return clearObjectAnnotation(ctx, c, &newWh, AnnotationKeyRefresh)
}
