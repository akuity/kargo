package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) CreateRole(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateRoleRequest],
) (*connect.Response[svcv1alpha1.CreateRoleResponse], error) {
	project := req.Msg.Role.Namespace
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := validateFieldNotEmpty("name", req.Msg.Role.Name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	role, err := s.rolesDB.Create(ctx, req.Msg.Role)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating Kargo Role %q in project %q: %w", req.Msg.Role.Name, req.Msg.Role.Namespace, err,
		)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateRoleResponse{
			Role: role,
		},
	), nil
}
