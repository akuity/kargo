package v1alpha1

import (
	"context"
	"time"

	"github.com/pkg/errors"
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
		return nil, errors.Wrapf(
			err,
			"error getting Warehouse %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
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
	if err := refreshObject(ctx, c, warehouse, time.Now); err != nil {
		return nil, errors.Wrap(err, "refresh")
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
	return clearRefreshObject(ctx, c, &newWh)
}
