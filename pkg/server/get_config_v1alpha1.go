package server

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) GetConfig(
	context.Context,
	*connect.Request[svcv1alpha1.GetConfigRequest],
) (*connect.Response[svcv1alpha1.GetConfigResponse], error) {
	resp := svcv1alpha1.GetConfigResponse{
		ArgocdShards:                  s.argoCDURLStore.GetShards(),
		SecretManagementEnabled:       s.cfg.SecretManagementEnabled,
		ClusterSecretsNamespace:       s.cfg.ClusterSecretNamespace,
		HasAnalysisRunLogsUrlTemplate: s.cfg.AnalysisRunLogURLTemplate != "",
	}
	return connect.NewResponse(&resp), nil
}
