package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
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
