package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) UpdateConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateConfigMapRequest],
) (*connect.Response[svcv1alpha1.UpdateConfigMapResponse], error) {
	if err := s.validateUpdateConfigMapRequest(ctx, req.Msg); err != nil {
		return nil, err
	}

	configMap := s.configMapToK8sConfigMap(configMap{
		systemLevel: req.Msg.SystemLevel,
		project:     req.Msg.Project,
		name:        req.Msg.Name,
		data:        req.Msg.Data,
		description: req.Msg.Description,
	})

	if err := s.client.Update(ctx, configMap); err != nil {
		return nil, fmt.Errorf("update configmap: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.UpdateConfigMapResponse{
		ConfigMap: configMap,
	}), nil
}

func (s *server) validateUpdateConfigMapRequest(
	ctx context.Context,
	req *svcv1alpha1.UpdateConfigMapRequest,
) error {
	if !req.SystemLevel && req.Project != "" {
		if err := s.validateProjectExists(ctx, req.Project); err != nil {
			return err
		}
	}

	if err := validateFieldNotEmpty("name", req.Name); err != nil {
		return err
	}

	if len(req.Data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			errors.New("ConfigMap data cannot be empty"))
	}

	return nil
}
