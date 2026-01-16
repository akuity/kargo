package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type configMap struct {
	systemLevel bool
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateConfigMap(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateConfigMapRequest],
) (*connect.Response[svcv1alpha1.CreateConfigMapResponse], error) {
	if err := s.validateCreateConfigMapRequest(ctx, req.Msg); err != nil {
		return nil, err
	}

	configMap := s.configMapToK8sConfigMap(configMap{
		systemLevel: req.Msg.SystemLevel,
		project:     req.Msg.Project,
		name:        req.Msg.Name,
		data:        req.Msg.Data,
		description: req.Msg.Description,
	})

	if err := s.client.Create(ctx, configMap); err != nil {
		return nil, fmt.Errorf("create configmap: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.CreateConfigMapResponse{
		ConfigMap: configMap,
	}), nil
}

func (s *server) validateCreateConfigMapRequest(
	ctx context.Context,
	req *svcv1alpha1.CreateConfigMapRequest,
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

func (s *server) configMapToK8sConfigMap(cm configMap) *corev1.ConfigMap {
	var namespace string
	if cm.systemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		namespace = cm.project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.name,
			Namespace: namespace,
			Annotations: map[string]string{
				kargoapi.AnnotationKeyDescription: cm.description,
			},
		},
		Data: cm.data,
	}
}
