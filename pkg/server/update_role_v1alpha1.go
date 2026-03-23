package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func (s *server) UpdateRole(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateRoleRequest],
) (*connect.Response[svcv1alpha1.UpdateRoleResponse], error) {
	project := req.Msg.Role.Namespace
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.Role.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	role, err := s.rolesDB.Update(ctx, req.Msg.Role)
	if err != nil {
		return nil, fmt.Errorf(
			"error updating Kargo Role %q in project %q: %w", name, project, err,
		)
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateRoleResponse{
			Role: role,
		},
	), nil
}

// @id UpdateRole
// @Summary Update a project-level Kargo Role virtual resource
// @Description Update a project-level Kargo Role virtual resource by updating
// @Description the underlying Kubernetes ServiceAccount, Role, and RoleBinding
// @Description resources.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param role path string true "Role name"
// @Param body body object true "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
// @Success 200 {object} object "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
// @Router /v1beta1/projects/{project}/roles/{role} [put]
func (s *server) updateRole(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("role")

	role := &rbacapi.Role{}
	if !bindJSONOrError(c, role) {
		return
	}

	// Ensure the role name in the URL matches the body (if provided in body)
	if role.Name != name {
		_ = c.Error(libhttp.ErrorStr(
			"name in body does not match role name in URL",
			http.StatusBadRequest,
		))
		return
	}

	// Ensure namespace in body matches project in URL (if provided in body)
	if role.Namespace != "" && role.Namespace != project {
		_ = c.Error(libhttp.ErrorStr(
			"namespace in body does not match project name in URL",
			http.StatusBadRequest,
		))
		return
	}

	role.KargoManaged = true

	updatedRole, err := s.rolesDB.Update(ctx, role)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, updatedRole)
}
