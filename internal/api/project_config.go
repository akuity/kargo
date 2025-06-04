package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// GetProjectConfig returns a pointer to the ProjectConfig resource specified by
// the provided name. If no such resource is found, nil is returned instead.
func GetProjectConfig(
	ctx context.Context,
	c client.Client,
	name string,
) (*kargoapi.ProjectConfig, error) {
	projectCfg := kargoapi.ProjectConfig{}
	if err := c.Get(
		ctx,
		types.NamespacedName{
			Namespace: name,
			Name:      name,
		},
		&projectCfg,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting ProjectConfig %q: %w", name, err)
	}
	return &projectCfg, nil
}
