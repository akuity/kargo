package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// @id DeleteProjectRepoCredentials
// @Summary Delete project-level repository credentials
// @Description Delete repository credentials from a project's namespace.
// @Tags Credentials, Repo Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param repo-credentials path string true "Credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/repo-credentials/{repo-credentials} [delete]
func (s *server) deleteProjectRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("repo-credentials")

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		&secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSharedRepoCredentials
// @Summary Delete shared repository credentials
// @Description Delete shared repository credentials.
// @Tags Credentials, Repo Credentials, Shared
// @Security BearerAuth
// @Param repo-credentials path string true "Credentials name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/shared/repo-credentials/{repo-credentials} [delete]
func (s *server) deleteSharedRepoCredentials(c *gin.Context) {
	if !s.requireSecretManagement(c) {
		return
	}

	ctx := c.Request.Context()
	name := c.Param("repo-credentials")

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{Namespace: s.cfg.SharedResourcesNamespace, Name: name},
		&secret,
	); err != nil {
		_ = c.Error(err)
		return
	}

	if err := validateRepoCredentialSecret(&secret); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.client.Delete(ctx, &secret); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
