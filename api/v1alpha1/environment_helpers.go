package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetEnv returns a pointer to the Environment resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetEnv(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*Environment, error) {
	env := Environment{}
	if err := c.Get(ctx, namespacedName, &env); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Environment %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &env, nil
}
