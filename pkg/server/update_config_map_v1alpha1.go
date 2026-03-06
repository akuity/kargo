package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func (s *server) UpdateConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateConfigMapRequest],
) (*connect.Response[svcv1alpha1.UpdateConfigMapResponse], error) {
	if err := s.validateUpdateConfigMapRequest(ctx, req.Msg); err != nil {
		return nil, err
	}

	configMap := s.configMapToK8sConfigMap(configMap{
		systemLevel: req.Msg.SystemLevel,
		project:     req.Msg.Project,
		name:        req.Msg.Name,
		data:        req.Msg.Data,
		description: req.Msg.Description,
	})

	if err := s.client.Update(ctx, configMap); err != nil {
		return nil, fmt.Errorf("update configmap: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.UpdateConfigMapResponse{
		ConfigMap: configMap,
	}), nil
}

func (s *server) validateUpdateConfigMapRequest(
	ctx context.Context,
	req *svcv1alpha1.UpdateConfigMapRequest,
) error {
	if !req.SystemLevel && req.Project != "" {
		if err := s.validateProjectExists(ctx, req.Project); err != nil {
			return err
		}
	}

	if err := validateFieldNotEmpty("name", req.Name); err != nil {
		return err
	}

	if len(req.Data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			errors.New("ConfigMap data cannot be empty"))
	}

	return nil
}

// updateConfigMapRequest is the request body for updating a ConfigMap.
type updateConfigMapRequest struct {
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"data"`
} // @name UpdateConfigMapRequest

// @id UpdateProjectConfigMap
// @Summary Replace a project-level ConfigMap
// @Description Replace a ConfigMap in a project's namespace. All existing data
// @Description is replaced. Returns the updated Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param configmap path string true "ConfigMap name"
// @Param body body updateConfigMapRequest true "ConfigMap"
// @Success 200 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/projects/{project}/configmaps/{configmap} [put]
func (s *server) updateProjectConfigMap(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("configmap")

	var req updateConfigMapRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if len(req.Data) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"ConfigMap data cannot be empty",
			http.StatusBadRequest,
		))
		return
	}

	configMapObj := corev1.ConfigMap{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: project, Name: name},
		&configMapObj,
	); err != nil {
		_ = c.Error(err)
		return
	}

	applyConfigMapUpdateToK8sConfigMap(&configMapObj, req)

	if err := s.client.Update(ctx, &configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &configMapObj)
}

// @id UpdateSystemConfigMap
// @Summary Replace a system-level ConfigMap
// @Description Replace a system-level ConfigMap. All existing data is replaced.
// @Description Returns the updated Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param configmap path string true "ConfigMap name"
// @Param body body updateConfigMapRequest true "ConfigMap"
// @Success 200 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/system/configmaps/{configmap} [put]
func (s *server) updateSystemConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	var req updateConfigMapRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if len(req.Data) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"ConfigMap data cannot be empty",
			http.StatusBadRequest,
		))
		return
	}

	configMapObj := corev1.ConfigMap{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.SystemResourcesNamespace,
			Name:      name,
		},
		&configMapObj,
	); err != nil {
		_ = c.Error(err)
		return
	}

	applyConfigMapUpdateToK8sConfigMap(&configMapObj, req)

	if err := s.client.Update(ctx, &configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &configMapObj)
}

// @id UpdateSharedConfigMap
// @Summary Replace a shared ConfigMap
// @Description Replace a shared ConfigMap. All existing data is replaced.
// @Description Returns the updated Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param configmap path string true "ConfigMap name"
// @Param body body updateConfigMapRequest true "ConfigMap"
// @Success 200 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/shared/configmaps/{configmap} [put]
func (s *server) updateSharedConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	var req updateConfigMapRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if len(req.Data) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			"ConfigMap data cannot be empty",
			http.StatusBadRequest,
		))
		return
	}

	configMapObj := corev1.ConfigMap{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.SharedResourcesNamespace,
			Name:      name,
		},
		&configMapObj,
	); err != nil {
		_ = c.Error(err)
		return
	}

	applyConfigMapUpdateToK8sConfigMap(&configMapObj, req)

	if err := s.client.Update(ctx, &configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &configMapObj)
}

func applyConfigMapUpdateToK8sConfigMap(configMapObj *corev1.ConfigMap, req updateConfigMapRequest) {
	// Set the description annotation if provided
	if req.Description != "" {
		if configMapObj.Annotations == nil {
			configMapObj.Annotations = make(map[string]string, 1)
		}
		configMapObj.Annotations[kargoapi.AnnotationKeyDescription] = req.Description
	} else {
		delete(configMapObj.Annotations, kargoapi.AnnotationKeyDescription)
	}

	// Replace the data with the new data
	configMapObj.Data = req.Data
}
