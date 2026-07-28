package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @id DeleteProjectAPIToken
// @Summary Delete a project-level API token
// @Description Delete a project-level API token from a project's namespace.
// @Tags Rbac, Credentials, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param apitoken path string true "API token name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/api-tokens/{apitoken} [delete]
func (s *server) deleteProjectAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("apitoken")

	if err := s.rolesDB.DeleteAPIToken(ctx, false, project, name); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @id DeleteSystemAPIToken
// @Summary Delete a system-level API token
// @Description Delete a system-level API token.
// @Tags Rbac, Credentials, System-Level
// @Security BearerAuth
// @Param apitoken path string true "API token name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/system/api-tokens/{apitoken} [delete]
func (s *server) deleteSystemAPIToken(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("apitoken")

	if err := s.rolesDB.DeleteAPIToken(ctx, true, "", name); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
