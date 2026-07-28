package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// @id DeleteProjectConfigMap
// @Summary Delete a project-level ConfigMap
// @Description Delete a ConfigMap from a project's namespace.
// @Tags Core, Generic Config, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param configmap path string true "ConfigMap name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/configmaps/{configmap} [delete]
func (s *server) deleteProjectConfigMap(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("configmap")

	configMapObj := &corev1.ConfigMap{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: project, Name: name},
		configMapObj,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSystemConfigMap
// @Summary Delete a system-level ConfigMap
// @Description Delete a system-level ConfigMap.
// @Tags Core, Generic Config, System-Level
// @Security BearerAuth
// @Param configmap path string true "ConfigMap name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/system/configmaps/{configmap} [delete]
func (s *server) deleteSystemConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	configMapObj := &corev1.ConfigMap{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.SystemResourcesNamespace,
			Name:      name,
		},
		configMapObj,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSharedConfigMap
// @Summary Delete a shared ConfigMap
// @Description Delete a shared ConfigMap.
// @Tags Core, Generic Config, Shared
// @Security BearerAuth
// @Param configmap path string true "ConfigMap name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/shared/configmaps/{configmap} [delete]
func (s *server) deleteSharedConfigMap(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("configmap")

	configMapObj := &corev1.ConfigMap{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.SharedResourcesNamespace,
			Name:      name,
		},
		configMapObj,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, configMapObj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
