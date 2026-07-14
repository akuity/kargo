package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @id DeleteProjectRole
// @Summary Delete a project-level Kargo Role virtual resource
// @Description Delete a project-level Kargo Role virtual resource by deleting
// @Description the underlying Kubernetes ServiceAccount, Role, and RoleBinding
// @Description resources from the project's namespace.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param role path string true "Role name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/roles/{role} [delete]
func (s *server) deleteProjectRole(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("role")

	if err := s.rolesDB.Delete(ctx, project, name); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
