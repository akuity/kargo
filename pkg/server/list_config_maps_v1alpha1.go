package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListConfigMaps(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListConfigMapsRequest],
) (*connect.Response[svcv1alpha1.ListConfigMapsResponse], error) {
	var namespace string
	if req.Msg.SystemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		project := req.Msg.Project
		if project != "" {
			if err := s.validateProjectExists(ctx, project); err != nil {
				return nil, err
			}
		}
		namespace = project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}

	var configMapsList corev1.ConfigMapList
	if err := s.client.List(
		ctx,
		&configMapsList,
		client.InNamespace(namespace),
	); err != nil {
		return nil, fmt.Errorf("list configmaps: %w", err)
	}

	configMaps := configMapsList.Items
	slices.SortFunc(configMaps, func(lhs, rhs corev1.ConfigMap) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	cmPtrs := []*corev1.ConfigMap{}
	for _, cm := range configMaps {
		cmPtrs = append(cmPtrs, cm.DeepCopy())
	}

	return connect.NewResponse(&svcv1alpha1.ListConfigMapsResponse{
		ConfigMaps: cmPtrs,
	}), nil
}

// @id ListProjectConfigMaps
// @Summary List project-level ConfigMaps
// @Description List ConfigMap resources from a project's namespace. Returns a
// @Description Kubernetes ConfigMapList resource.
// @Tags Core, Generic Config, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} object "ConfigMapList resource (k8s.io/api/core/v1.ConfigMapList)"
// @Router /v1beta1/projects/{project}/configmaps [get]
func (s *server) listProjectConfigMaps(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	list := &corev1.ConfigMapList{}
	if err := s.client.List(
		ctx, list, client.InNamespace(project),
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.ConfigMap) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}

// @id ListSystemConfigMaps
// @Summary List system-level ConfigMaps
// @Description List system-level ConfigMap resources. Returns a Kubernetes
// @Description ConfigMapList resource.
// @Tags Core, Generic Config, System-Level
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "ConfigMapList resource (k8s.io/api/core/v1.ConfigMapList)"
// @Router /v1beta1/system/configmaps [get]
func (s *server) listSystemConfigMaps(c *gin.Context) {
	ctx := c.Request.Context()

	list := &corev1.ConfigMapList{}
	if err := s.client.List(
		ctx, list, client.InNamespace(s.cfg.SystemResourcesNamespace),
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.ConfigMap) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}

// @id ListSharedConfigMaps
// @Summary List shared ConfigMaps
// @Description List shared ConfigMap resources referenceable by all projects.
// @Description Returns a Kubernetes ConfigMapList resource.
// @Tags Core, Generic Config, Shared
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "ConfigMapList resource (k8s.io/api/core/v1.ConfigMapList)"
// @Router /v1beta1/shared/configmaps [get]
func (s *server) listSharedConfigMaps(c *gin.Context) {
	ctx := c.Request.Context()

	list := &corev1.ConfigMapList{}
	if err := s.client.List(
		ctx, list, client.InNamespace(s.cfg.SharedResourcesNamespace),
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.ConfigMap) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}
