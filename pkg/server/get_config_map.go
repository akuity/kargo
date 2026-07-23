package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// @id GetProjectConfigMap
// @Summary Retrieve a project-level ConfigMap
// @Description Retrieve a ConfigMap by name from a project's namespace.
// @Tags Core, Generic Config, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param configmap path string true "ConfigMap name"
// @Produce json
// @Success 200 {object} corev1.ConfigMap "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/projects/{project}/configmaps/{configmap} [get]
func (s *server) getProjectConfigMap(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("configmap")

	configMap := &corev1.ConfigMap{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: project}, configMap,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, configMap)
}

// @id GetSystemConfigMap
// @Summary Retrieve a system-level ConfigMap
// @Description Retrieve a system-level ConfigMap by name.
// @Tags Core, Generic Config, System-Level
// @Security BearerAuth
// @Param configmap path string true "ConfigMap name"
// @Produce json
// @Success 200 {object} corev1.ConfigMap "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/system/configmaps/{configmap} [get]
func (s *server) getSystemConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	configMap := &corev1.ConfigMap{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: s.cfg.SystemResourcesNamespace}, configMap,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, configMap)
}

// @id GetSharedConfigMap
// @Summary Retrieve a shared ConfigMap
// @Description Retrieve a shared ConfigMap by name.
// @Tags Core, Generic Config, Shared
// @Security BearerAuth
// @Param configmap path string true "ConfigMap name"
// @Produce json
// @Success 200 {object} corev1.ConfigMap "ConfigMap resource (k8s.io/api/core/v1.ConfigMap)"
// @Router /v1beta1/shared/configmaps/{configmap} [get]
func (s *server) getSharedConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	configMap := &corev1.ConfigMap{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: s.cfg.SharedResourcesNamespace}, configMap,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, configMap)
}
