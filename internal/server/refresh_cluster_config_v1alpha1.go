package server

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func (s *server) RefreshClusterConfig(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.RefreshClusterConfigRequest],
) (*connect.Response[svcv1alpha1.RefreshClusterConfigResponse], error) {
	config, err := api.RefreshClusterConfig(ctx, s.client.InternalClient())
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&svcv1alpha1.RefreshClusterConfigResponse{
		ClusterConfig: config,
	}), nil
}
