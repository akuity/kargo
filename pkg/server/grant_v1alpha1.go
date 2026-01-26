package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func (s *server) Grant(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GrantRequest],
) (*connect.Response[svcv1alpha1.GrantResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var role *rbacapi.Role
	var err error
	if userClaims := req.Msg.GetUserClaims(); userClaims != nil {
		claims := make([]rbacapi.Claim, len(userClaims.Claims))
		for i, claim := range userClaims.Claims {
			claims[i] = *claim
		}
		if role, err = s.rolesDB.GrantRoleToUsers(
			ctx, project, req.Msg.Role, claims,
		); err != nil {
			return nil, fmt.Errorf("error granting Kargo Role to users: %w", err)
		}
	} else if resources := req.Msg.GetResourceDetails(); resources != nil {
		if role, err = s.rolesDB.GrantPermissionsToRole(
			ctx, project, req.Msg.Role, resources,
		); err != nil {
			return nil, fmt.Errorf("error granting permissions to Kargo Role: %w", err)
		}
	} else {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("one of userClaims or resourceDetails must be provided"),
		)
	}

	return connect.NewResponse(
		&svcv1alpha1.GrantResponse{
			Role: role,
		},
	), nil
}

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
// @Success 200 {object} object "Role resource (github.com/akuity/kargo/api/rbac/v1alpha1.Role)"
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
