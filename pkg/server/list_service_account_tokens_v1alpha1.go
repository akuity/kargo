package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListServiceAccountTokens(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListServiceAccountTokensRequest],
) (*connect.Response[svcv1alpha1.ListServiceAccountTokensResponse], error) {
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

	serviceAccountName := req.Msg.ServiceAccountName

	tokenSecrets, err := s.serviceAccountsDB.ListTokens(
		ctx, systemLevel, project, serviceAccountName,
	)
	if err != nil {
		if serviceAccountName == "" {
			return nil, fmt.Errorf(
				"error listing Kargo ServiceAccount tokens in project %q: %w",
				project, err,
			)
		}
		return nil, fmt.Errorf(
			"error listing tokens for Kargo ServiceAccount %q in project %q: %w",
			serviceAccountName, project, err,
		)
	}
	secretPtrs := make([]*corev1.Secret, len(tokenSecrets))
	for i, tokenSecret := range tokenSecrets {
		secretPtrs[i] = &tokenSecret
	}

	return connect.NewResponse(&svcv1alpha1.ListServiceAccountTokensResponse{
		TokenSecrets: secretPtrs,
	}), nil
}
