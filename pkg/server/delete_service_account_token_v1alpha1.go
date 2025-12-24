package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) DeleteServiceAccountToken(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteServiceAccountTokenRequest],
) (*connect.Response[svcv1alpha1.DeleteServiceAccountTokenResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	if err := s.serviceAccountsDB.DeleteToken(
		ctx, systemLevel, project, name,
	); err != nil {
		return nil, fmt.Errorf("error deleting ServiceAccount token Secret: %w", err)
	}
	return connect.NewResponse(
		&svcv1alpha1.DeleteServiceAccountTokenResponse{},
	), nil
}
