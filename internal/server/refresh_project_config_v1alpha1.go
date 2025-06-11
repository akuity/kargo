package server

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func (s *server) RefreshProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshProjectConfigRequest],
) (*connect.Response[svcv1alpha1.RefreshProjectConfigResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	config, err := api.RefreshProjectConfig(ctx, s.client.InternalClient(), project)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.RefreshProjectConfigResponse{
		ProjectConfig: config,
	}), nil
}
