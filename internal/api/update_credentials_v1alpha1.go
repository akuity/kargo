package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdateCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateCredentialsRequest],
) (*connect.Response[svcv1alpha1.UpdateCredentialsResponse], error) {
	creds := credentials{
		project:        req.Msg.GetProject(),
		name:           req.Msg.GetName(),
		credType:       req.Msg.GetType(),
		repoURL:        req.Msg.GetRepoUrl(),
		repoURLPattern: req.Msg.GetRepoUrlPattern(),
		username:       req.Msg.GetUsername(),
		password:       req.Msg.GetPassword(),
	}

	if err := s.validateCredentials(creds); err != nil {
		return nil, err
	}

	if err := s.client.Update(ctx, credentialsToSecret(creds)); err != nil {
		return nil, errors.Wrap(err, "update secret")
	}

	return connect.NewResponse(&svcv1alpha1.UpdateCredentialsResponse{}), nil
}
