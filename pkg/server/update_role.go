package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

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
// @Success 200 {object} rbacapi.Role "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
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
