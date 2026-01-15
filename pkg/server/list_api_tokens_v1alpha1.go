package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListAPITokens(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListAPITokensRequest],
) (*connect.Response[svcv1alpha1.ListAPITokensResponse], error) {
	systemLevel := req.Msg.SystemLevel
	project := req.Msg.Project
	if err := s.validateSystemLevelOrProject(systemLevel, project); err != nil {
		return nil, err
	}

	if !systemLevel {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}

	roleName := req.Msg.RoleName

	tokenSecrets, err := s.rolesDB.ListAPITokens(ctx, systemLevel, project, roleName)
	if err != nil {
		if roleName == "" {
			return nil, fmt.Errorf(
				"error listing Kargo API tokens in project %q: %w",
				project, err,
			)
		}
		return nil, fmt.Errorf(
			"error listing tokens for Kargo API role %q in project %q: %w",
			roleName, project, err,
		)
	}
	secretPtrs := make([]*svcv1alpha1.Secret, len(tokenSecrets))
	for i, tokenSecret := range tokenSecrets {
		secretPtrs[i] = svcv1alpha1.FromK8sSecret(&tokenSecret)
	}

	return connect.NewResponse(
		&svcv1alpha1.ListAPITokensResponse{TokenSecrets: secretPtrs},
	), nil
}
