package v1alpha1

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetProject returns a pointer to the cluster-scoped Project resource specified
// by the name argument. If no such resource is found, nil is returned instead.
func GetProject(
	ctx context.Context,
	c client.Client,
	name string,
) (*Project, error) {
	project := Project{}
	if err := c.Get(
		ctx, types.NamespacedName{
			Name: name,
		},
		&project,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error getting Project %q", name)
	}
	return &project, nil
}
