package api

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetConfig(
	context.Context,
	*connect.Request[svcv1alpha1.GetConfigRequest],
) (*connect.Response[svcv1alpha1.GetConfigResponse], error) {
	resp := svcv1alpha1.GetConfigResponse{
		ArgocdShards: make(map[string]*svcv1alpha1.ArgoCDShard),
	}
	for shardName, url := range s.cfg.ArgoCDConfig.URLs {
		resp.ArgocdShards[shardName] = &svcv1alpha1.ArgoCDShard{
			Url: url,
			// TODO: currently, all shards must use the same namespace
			Namespace: s.cfg.ArgoCDConfig.Namespace,
		}
	}
	return connect.NewResponse(&resp), nil
}
