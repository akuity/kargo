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

// RefreshProjectConfig forces reconciliation the ProjectConfig by setting an
// annotation on the ProjectConfig, causing the controller to reconcile it.
// Currently, the annotation value is the timestamp of the request, but might in
// the future include additional metadata/context necessary for the request.
func RefreshProjectConfig(
	ctx context.Context,
	c client.Client,
	project string,
) (*kargoapi.ProjectConfig, error) {
	config := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project,
			Namespace: project,
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
