package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const ClusterConfigName = "cluster"

// GetClusterConfig returns a pointer to the ClusterConfig resource. If no such
// resource is found, nil is returned instead.
func GetClusterConfig(
	ctx context.Context,
	c client.Client,
) (*kargoapi.ClusterConfig, error) {
	clusterCfg := kargoapi.ClusterConfig{}
	if err := c.Get(
		ctx, types.NamespacedName{Name: ClusterConfigName},
		&clusterCfg,
	); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting ClusterConfig: %w", err)
	}
	return &clusterCfg, nil
}
