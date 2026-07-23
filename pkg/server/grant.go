package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// grantRequest represents the request body for the Grant REST endpoint.
type grantRequest struct {
	Role            string                   `json:"role"`
	UserClaims      *userClaims              `json:"userClaims,omitempty"`
	ResourceDetails *rbacapi.ResourceDetails `json:"resourceDetails,omitempty"`
} // @name GrantRequest

// userClaims represents a list of claims for users.
type userClaims struct {
	Claims []rbacapi.Claim `json:"claims"`
} // @name UserClaims

// @id Grant
// @Summary Grant permissions
// @Description Grant a project-level Kargo Role to users or grant permissions
// @Description to a project-level Kargo Role.
// @Tags Rbac, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body grantRequest true "Grant request"
// @Success 200 {object} rbacapi.Role "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
// @Router /v1beta1/projects/{project}/roles/grants [post]
func (s *server) grant(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	var req grantRequest
	if !bindJSONOrError(c, &req) {
		return
	}

	if req.Role == "" {
		_ = c.Error(libhttp.ErrorStr("role is required", http.StatusBadRequest))
		return
	}

	var role *rbacapi.Role
	var err error

	switch {
	case req.UserClaims != nil:
		role, err = s.rolesDB.GrantRoleToUsers(ctx, project, req.Role, req.UserClaims.Claims)
		if err != nil {
			_ = c.Error(err)
			return
		}
	case req.ResourceDetails != nil:
		role, err = s.rolesDB.GrantPermissionsToRole(ctx, project, req.Role, req.ResourceDetails)
		if err != nil {
			_ = c.Error(err)
			return
		}
	default:
		_ = c.Error(libhttp.ErrorStr(
			"one of userClaims, serviceAccounts, or resourceDetails must be provided",
			http.StatusBadRequest,
		))
		return
	}

	c.JSON(http.StatusOK, role)
}
