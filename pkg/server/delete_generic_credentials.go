package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// @id DeleteProjectGenericCredentials
// @Summary Delete project-level generic credentials
// @Description Delete generic credentials from a project's namespace.
// @Tags Credentials, Generic Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param generic-credentials path string true "Generic credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/generic-credentials/{generic-credentials} [delete]
func (s *server) deleteProjectGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("generic-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: project, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSystemGenericCredentials
// @Summary Delete system-level generic credentials
// @Description Delete system-level generic credentials.
// @Tags Credentials, Generic Credentials, System-Level
// @Security BearerAuth
// @Param generic-credentials path string true "Generic credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/system/generic-credentials/{generic-credentials} [delete]
func (s *server) deleteSystemGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("generic-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: s.cfg.SystemResourcesNamespace, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSharedGenericCredentials
// @Summary Delete shared generic credentials
// @Description Delete shared generic credentials.
// @Tags Credentials, Generic Credentials, Shared
// @Security BearerAuth
// @Param generic-credentials path string true "Generic credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/shared/generic-credentials/{generic-credentials} [delete]
func (s *server) deleteSharedGenericCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("generic-credentials")

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: s.cfg.SharedResourcesNamespace, Name: name},
		secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateGenericCredentialSecret(secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
