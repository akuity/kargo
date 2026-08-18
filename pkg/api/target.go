package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GetTarget returns a pointer to the Target resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetTarget(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Target, error) {
	target := kargoapi.Target{}
	if err := c.Get(ctx, namespacedName, &target); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Target %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &target, nil
}
