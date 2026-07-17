package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
)

// @id ListProjectAPITokens
// @Summary List project-level API tokens
// @Description List project-level API tokens. Returns a Kubernetes SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param role query string false "Role name filter"
// @Produce json
// @Success 200 {object} corev1.SecretList "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/projects/{project}/api-tokens [get]
func (s *server) listProjectAPITokens(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	roleName := c.Query("role")

	tokens, err := s.rolesDB.ListAPITokens(ctx, false, project, roleName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, corev1.SecretList{Items: tokens})
}

// @id ListSystemAPITokens
// @Summary List system-level API tokens
// @Description List system-level API tokens. Returns a Kubernetes SecretList
// @Description resource containing heavily redacted Secrets.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param role query string false "Role name filter"
// @Produce json
// @Success 200 {object} corev1.SecretList "SecretList resource (k8s.io/api/core/v1.SecretList)"
// @Router /v1beta1/system/api-tokens [get]
func (s *server) listSystemAPITokens(c *gin.Context) {
	ctx := c.Request.Context()

	roleName := c.Query("role")

	tokens, err := s.rolesDB.ListAPITokens(ctx, true, "", roleName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, corev1.SecretList{Items: tokens})
}
