package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) DeleteRole(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteRoleRequest],
) (*connect.Response[svcv1alpha1.DeleteRoleResponse], error) {
	project := req.Msg.Project
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	if err := s.rolesDB.Delete(ctx, project, name); err != nil {
		return nil, fmt.Errorf(
			"error deleting Kargo Role %q in project %q: %w", name, project, err,
		)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteRoleResponse{}), nil
}

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
