package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) CreateAPIToken(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateAPITokenRequest],
) (*connect.Response[svcv1alpha1.CreateAPITokenResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	roleName := req.Msg.RoleName
	if err := validateFieldNotEmpty("role_name", roleName); err != nil {
		return nil, err
	}

	name := req.Msg.Name
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	tokenSecret, err := s.rolesDB.CreateAPIToken(
		ctx, systemLevel, project, roleName, name,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating new token Secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateAPITokenResponse{TokenSecret: svcv1alpha1.FromK8sSecret(tokenSecret)},
	), nil
}
