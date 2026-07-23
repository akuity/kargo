package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id CreateProjectRole
// @Summary Create a project-level Kargo Role virtual resource
// @Description Create a project-level Kargo Role virtual resource by creating
// @Description the underlying Kubernetes ServiceAccount, Role, and RoleBinding
// @Description resources.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body object true "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
// @Success 201 {object} rbacapi.Role "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
// @Router /v1beta1/projects/{project}/roles [post]
func (s *server) createProjectRole(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	role := &rbacapi.Role{}
	if !bindJSONOrError(c, role) {
		return
	}

	if role.Name == "" {
		_ = c.Error(libhttp.Error(
			errors.New("name should not be empty"),
			http.StatusBadRequest,
		))
		return
	}

	// Ensure namespace in body matches project in URL (if provided in body)
	if role.Namespace != "" && role.Namespace != project {
		_ = c.Error(libhttp.Error(
			errors.New("namespace in body does not match project in URL"),
			http.StatusConflict,
		))
		return
	}

	// Set namespace from URL path parameter
	role.Namespace = project
	role.KargoManaged = true

	createdRole, err := s.rolesDB.Create(ctx, role)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, createdRole)
}
