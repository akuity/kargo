package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GetProject returns a pointer to the cluster-scoped Project resource specified
// by the name argument. If no such resource is found, nil is returned instead.
func GetProject(
	ctx context.Context,
	c client.Client,
	name string,
) (*kargoapi.Project, error) {
	project := kargoapi.Project{}
	if err := c.Get(
		ctx, types.NamespacedName{
			Name: name,
		},
		&project,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting Project %q: %w", name, err)
	}
	return &project, nil
}
