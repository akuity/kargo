package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
)

// @id GetProjectAPIToken
// @Summary Retrieve a project-level API token
// @Description Retrieve a project-level API token by name. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param apitoken path string true "API token name"
// @Produce json
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/projects/{project}/api-tokens/{apitoken} [get]
func (s *server) getProjectAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("apitoken")

	var tokenSecret *corev1.Secret
	tokenSecret, err := s.rolesDB.GetAPIToken(ctx, false, project, name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenSecret)
}

// @id GetSystemAPIToken
// @Summary Retrieve a system-level API token
// @Description Retrieve a system-level API token by name. Returns a heavily
// @Description redacted Kubernetes Secret resource.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param apitoken path string true "API token name"
// @Produce json
// @Success 200 {object} corev1.Secret "Secret resource (k8s.io/api/core/v1.Secret)"
// @Router /v1beta1/system/api-tokens/{apitoken} [get]
func (s *server) getSystemAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("apitoken")

	tokenSecret, err := s.rolesDB.GetAPIToken(ctx, true, "", name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, tokenSecret)
}
