package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) DeleteServiceAccount(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteServiceAccountRequest],
) (*connect.Response[svcv1alpha1.DeleteServiceAccountResponse], error) {
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

	if err := s.serviceAccountsDB.Delete(ctx, project, name); err != nil {
		return nil, fmt.Errorf(
			"error deleting Kargo ServiceAccount %q in project %q: %w",
			name, project, err,
		)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteServiceAccountResponse{}), nil
}
