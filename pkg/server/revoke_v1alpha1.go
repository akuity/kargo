package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) Revoke(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RevokeRequest],
) (*connect.Response[svcv1alpha1.RevokeResponse], error) {
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
		if role, err = s.rolesDB.RevokeRoleFromUsers(
			ctx, project, req.Msg.Role, claims,
		); err != nil {
			return nil, fmt.Errorf("error revoking Kargo Role from users: %w", err)
		}
	} else if serviceAccounts := req.Msg.GetServiceAccounts(); serviceAccounts != nil {
		saRefs := make([]rbacapi.ServiceAccountReference, len(serviceAccounts.ServiceAccounts))
		for i, saRef := range serviceAccounts.ServiceAccounts {
			saRefs[i] = *saRef
		}
		if role, err = s.rolesDB.RevokeRoleFromServiceAccounts(
			ctx, project, req.Msg.Role, saRefs,
		); err != nil {
			return nil,
				fmt.Errorf("error revoking Kargo Role from ServiceAccounts: %w", err)
		}
	} else if resources := req.Msg.GetResourceDetails(); resources != nil {
		if role, err = s.rolesDB.RevokePermissionsFromRole(
			ctx, project, req.Msg.Role, resources,
		); err != nil {
			return nil, fmt.Errorf("error revoking permissions from Kargo Role: %w", err)
		}
	} else {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New(
				"one of userClaims, serviceAccount, or resourceDetails must be "+
					"provided",
			),
		)
	}

	return connect.NewResponse(
		&svcv1alpha1.RevokeResponse{
			Role: role,
		},
	), nil
}
