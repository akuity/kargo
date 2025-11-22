package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListServiceAccounts(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListServiceAccountsRequest],
) (*connect.Response[svcv1alpha1.ListServiceAccountsResponse], error) {
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

	saList, err := s.serviceAccountsDB.List(ctx, systemLevel, project)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Kargo ServiceAccounts in project %q: %w", project, err,
		)
	}
	saPtrList := make([]*corev1.ServiceAccount, len(saList))
	for i, sa := range saList {
		saPtrList[i] = &sa
	}

	return connect.NewResponse(&svcv1alpha1.ListServiceAccountsResponse{
		ServiceAccounts: saPtrList,
	}), nil
}
