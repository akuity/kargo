package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
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
