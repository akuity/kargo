package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) UpdateClusterSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateClusterSecretRequest],
) (*connect.Response[svcv1alpha1.UpdateClusterSecretResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errClusterSecretNamespaceNotDefined)
	}

	if s.cfg.ClusterSecretNamespace == "" {
		return nil, connect.NewError(connect.CodeUnimplemented, errClusterSecretNamespaceNotDefined)
	}

	name := req.Msg.GetName()

	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.ClusterSecretNamespace,
			Name:      name,
		},
		&secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	clusterSecretUpdate := clusterSecret{
		data: req.Msg.GetData(),
	}

	if len(clusterSecretUpdate.data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
	}

	applyClusterSecretUpdateToK8sSecret(&secret, clusterSecretUpdate)

	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateClusterSecretResponse{
			Secret: sanitizeProjectSecret(secret),
		},
	), nil
}

func applyClusterSecretUpdateToK8sSecret(secret *corev1.Secret, clusterSecretUpdate clusterSecret) {
	// Delete any keys in the secret that are not in the update
	for key := range secret.Data {
		if _, ok := clusterSecretUpdate.data[key]; !ok {
			delete(secret.Data, key)
		}
	}

	// Add or update the keys in the secret with the values from the update
	if secret.Data == nil {
		secret.Data = make(map[string][]byte, len(clusterSecretUpdate.data))
	}

	for key, value := range clusterSecretUpdate.data {
		if value != "" {
			secret.Data[key] = []byte(value)
		}
	}
}
