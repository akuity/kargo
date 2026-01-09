package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) DeleteSystemSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteSystemSecretRequest],
) (*connect.Response[svcv1alpha1.DeleteSystemSecretResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: s.cfg.SystemResourcesNamespace,
			Name:      name,
		},
		secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	if err := s.client.Delete(ctx, secret); err != nil {
		return nil, fmt.Errorf("delete secret: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteSystemSecretResponse{}), nil
}
