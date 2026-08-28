package server

import (
	"context"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func (s *server) RefreshClusterConfig(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.RefreshClusterConfigRequest],
) (*connect.Response[svcv1alpha1.RefreshClusterConfigResponse], error) {
	key := client.ObjectKey{Name: api.ClusterConfigName}
	if err := s.authorizeFn(
		ctx, "get", kargoapi.GroupVersion.WithResource("clusterconfigs"), "", key,
	); err != nil {
		return nil, err
	}

	config, err := api.RefreshClusterConfig(ctx, s.client.InternalClient())
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	return connect.NewResponse(&svcv1alpha1.RefreshClusterConfigResponse{
		ClusterConfig: config,
	}), nil
}
