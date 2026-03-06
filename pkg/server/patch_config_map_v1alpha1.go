package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

var errEmptyConfigMap = errors.New("ConfigMap data cannot be empty after patch")

// patchConfigMapRequest is the request body for patching a ConfigMap.
// All fields are optional. Provided data is merged with existing data.
// Use removeKeys to delete specific keys.
type patchConfigMapRequest struct {
	Description *string           `json:"description,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	RemoveKeys  []string          `json:"removeKeys,omitempty"`
} // @name PatchConfigMapRequest

// @id PatchProjectConfigMap
// @Summary Patch a project-level ConfigMap
// @Description Patch a ConfigMap in a project's namespace. Merges provided data
// @Description with existing data. Use removeKeys to delete specific keys.
// @Description Returns the updated Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param configmap path string true "ConfigMap name"
// @Param body body patchConfigMapRequest true "ConfigMap patch"
// @Success 200 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/projects/{project}/configmaps/{configmap} [patch]
func (s *server) patchProjectConfigMap(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("configmap")

	var req patchConfigMapRequest
	if !bindJSONOrError(c, &req) {
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

	applyConfigMapPatchToK8sConfigMap(&configMapObj, req)

	if err := validateConfigMapNotEmpty(&configMapObj); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	if err := s.client.Update(ctx, &configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &configMapObj)
}

// @id PatchSystemConfigMap
// @Summary Patch a system-level ConfigMap
// @Description Patch a system-level ConfigMap. Merges provided data
// @Description with existing data. Use removeKeys to delete specific keys.
// @Description Returns the updated Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, System-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param configmap path string true "ConfigMap name"
// @Param body body patchConfigMapRequest true "ConfigMap patch"
// @Success 200 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/system/configmaps/{configmap} [patch]
func (s *server) patchSystemConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	var req patchConfigMapRequest
	if !bindJSONOrError(c, &req) {
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

	applyConfigMapPatchToK8sConfigMap(&configMapObj, req)

	if err := validateConfigMapNotEmpty(&configMapObj); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	if err := s.client.Update(ctx, &configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &configMapObj)
}

// @id PatchSharedConfigMap
// @Summary Patch a shared ConfigMap
// @Description Patch a shared ConfigMap. Merges provided data
// @Description with existing data. Use removeKeys to delete specific keys.
// @Description Returns the updated Kubernetes ConfigMap resource.
// @Tags Core, Generic Config, Shared
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param configmap path string true "ConfigMap name"
// @Param body body patchConfigMapRequest true "ConfigMap patch"
// @Success 200 {object} object "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/shared/configmaps/{configmap} [patch]
func (s *server) patchSharedConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	var req patchConfigMapRequest
	if !bindJSONOrError(c, &req) {
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

	applyConfigMapPatchToK8sConfigMap(&configMapObj, req)

	if err := validateConfigMapNotEmpty(&configMapObj); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	if err := s.client.Update(ctx, &configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &configMapObj)
}

func applyConfigMapPatchToK8sConfigMap(configMapObj *corev1.ConfigMap, req patchConfigMapRequest) {
	// Update description if provided (nil means don't change, empty string means clear)
	if req.Description != nil {
		if *req.Description != "" {
			if configMapObj.Annotations == nil {
				configMapObj.Annotations = make(map[string]string, 1)
			}
			configMapObj.Annotations[kargoapi.AnnotationKeyDescription] = *req.Description
		} else {
			delete(configMapObj.Annotations, kargoapi.AnnotationKeyDescription)
		}
	}

	// Remove specified keys
	for _, key := range req.RemoveKeys {
		delete(configMapObj.Data, key)
	}

	// Merge new data (add or update keys)
	if configMapObj.Data == nil {
		configMapObj.Data = make(map[string]string, len(req.Data))
	}
	for key, value := range req.Data {
		configMapObj.Data[key] = value
	}
}

func validateConfigMapNotEmpty(configMapObj *corev1.ConfigMap) error {
	if len(configMapObj.Data) == 0 {
		return errEmptyConfigMap
	}
	return nil
}
