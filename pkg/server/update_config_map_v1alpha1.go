package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) UpdateConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateConfigMapRequest],
) (*connect.Response[svcv1alpha1.UpdateConfigMapResponse], error) {
	configMap := req.Msg.ConfigMap
	if configMap == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("config_map is required"))
	}

	if err := validateFieldNotEmpty("name", configMap.Name); err != nil {
		return nil, err
	}

	var namespace string
	if req.Msg.SystemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		project := configMap.Namespace
		if project != "" {
			if err := s.validateProjectExists(ctx, project); err != nil {
				return nil, err
			}
		}
		namespace = project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}

	configMap.Namespace = namespace

	if err := s.client.Update(ctx, configMap); err != nil {
		return nil, fmt.Errorf("update configmap: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.UpdateConfigMapResponse{
		ConfigMap: configMap,
	}), nil
}
