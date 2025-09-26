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

type clusterSecret struct {
	name string
	data map[string]string
}

func (s *server) CreateClusterSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateClusterSecretRequest],
) (*connect.Response[svcv1alpha1.CreateClusterSecretResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	if s.cfg.ClusterSecretNamespace == "" {
		return nil, connect.NewError(connect.CodeUnimplemented, errClusterSecretNamespaceNotDefined)
	}

	clsSecret := clusterSecret{
		name: req.Msg.GetName(),
		data: req.Msg.GetData(),
	}

	if err := s.validateClusterSecret(clsSecret); err != nil {
		return nil, err
	}

	secret := s.clusterSecretToK8sSecret(clsSecret)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateClusterSecretResponse{
			Secret: sanitizeProjectSecret(*secret),
		},
	), nil
}

func (s *server) validateClusterSecret(clsSecret clusterSecret) error {
	if err := validateFieldNotEmpty("name", clsSecret.name); err != nil {
		return err
	}

	if len(clsSecret.data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
	}

	return nil
}

func (s *server) clusterSecretToK8sSecret(clsSecret clusterSecret) *corev1.Secret {
	secretsData := map[string][]byte{}

	for key, value := range clsSecret.data {
		secretsData[key] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.cfg.ClusterSecretNamespace,
			Name:      clsSecret.name,
		},
		Data: secretsData,
	}

	return secret
}
