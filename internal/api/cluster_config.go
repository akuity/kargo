package api

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// RefreshClusterConfig forces reconciliation the ClusterConfig by setting an
// annotation on the ClusterConfig, causing the controller to reconcile it.
// Currently, the annotation value is the timestamp of the request, but might in
// the future include additional metadata/context necessary for the request.
func RefreshClusterConfig(
	ctx context.Context,
	c client.Client,
) (*kargoapi.ClusterConfig, error) {
	config := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClusterConfigName,
		},
	}
	if err := patchAnnotation(
		ctx,
		c,
		config,
		kargoapi.AnnotationKeyRefresh,
		time.Now().Format(time.RFC3339),
	); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return config, nil
}
