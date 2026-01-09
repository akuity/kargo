package server

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
)

func (s *server) ListSystemSecrets(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListSystemSecretsRequest],
) (*connect.Response[svcv1alpha1.ListSystemSecretsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	var secretsList corev1.SecretList
	if err := s.client.List(
		ctx,
		&secretsList,
		client.InNamespace(s.cfg.SystemResourcesNamespace),
	); err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	// Sort the secrets by name
	secrets := secretsList.Items
	slices.SortFunc(secrets, func(lhs, rhs corev1.Secret) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	sanitizedSecrets := make([]*corev1.Secret, len(secrets))
	for i, secret := range secrets {
		sanitizedSecrets[i] = sanitizeProjectSecret(secret)
	}

	return connect.NewResponse(&svcv1alpha1.ListSystemSecretsResponse{
		Secrets: sanitizedSecrets,
	}), nil
}
