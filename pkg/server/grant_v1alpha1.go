package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
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
			errors.New("either userClaims or resourceDetails must be provided"),
		)
	}

	return connect.NewResponse(
		&svcv1alpha1.GrantResponse{
			Role: role,
		},
	), nil
}
