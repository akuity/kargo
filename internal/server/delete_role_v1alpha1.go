package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
