package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

type systemSecret struct {
	name string
	data map[string]string
}

func (s *server) CreateSystemSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateSystemSecretRequest],
) (*connect.Response[svcv1alpha1.CreateSystemSecretResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	clsSecret := systemSecret{
		name: req.Msg.GetName(),
		data: req.Msg.GetData(),
	}

	if err := s.validateSystemSecret(clsSecret); err != nil {
		return nil, err
	}

	secret := s.systemSecretToK8sSecret(clsSecret)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateSystemSecretResponse{
			Secret: sanitizeProjectSecret(*secret),
		},
	), nil
}

func (s *server) validateSystemSecret(clsSecret systemSecret) error {
	if err := validateFieldNotEmpty("name", clsSecret.name); err != nil {
		return err
	}

	if len(clsSecret.data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
	}

	return nil
}

func (s *server) systemSecretToK8sSecret(clsSecret systemSecret) *corev1.Secret {
	secretsData := map[string][]byte{}

	for key, value := range clsSecret.data {
		secretsData[key] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.cfg.SystemResourcesNamespace,
			Name:      clsSecret.name,
		},
		Data: secretsData,
	}

	return secret
}
