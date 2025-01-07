package api

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteCredentialsRequest],
) (*connect.Response[svcv1alpha1.DeleteCredentialsResponse], error) {
	_, err := s.DeleteSecrets(ctx, &connect.Request[svcv1alpha1.DeleteSecretsRequest]{
		Msg: &svcv1alpha1.DeleteSecretsRequest{
			Project: req.Msg.GetProject(),
			Name:    req.Msg.GetName(),
		},
	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&svcv1alpha1.DeleteCredentialsResponse{}), nil
}
