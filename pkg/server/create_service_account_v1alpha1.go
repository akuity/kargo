package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) CreateServiceAccount(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateServiceAccountRequest],
) (*connect.Response[svcv1alpha1.CreateServiceAccountResponse], error) {
	project := req.Msg.ServiceAccount.Namespace
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := validateFieldNotEmpty("name", req.Msg.ServiceAccount.Name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	sa, err := s.serviceAccountsDB.Create(ctx, req.Msg.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating Kargo ServiceAccount %q in project %q: %w",
			req.Msg.ServiceAccount.Name, req.Msg.ServiceAccount.Namespace, err,
		)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateServiceAccountResponse{
			ServiceAccount: sa,
		},
	), nil
}
