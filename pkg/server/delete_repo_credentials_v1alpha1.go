package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) DeleteRepoCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteRepoCredentialsRequest],
) (*connect.Response[svcv1alpha1.DeleteRepoCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	project := req.Msg.GetProject()
	if project != "" {
		if err := s.validateProjectExists(ctx, project); err != nil {
			return nil, err
		}
	}
	namespace := project
	if namespace == "" {
		namespace = s.cfg.SharedResourcesNamespace
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		&secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	// If this isn't labeled as repository credentials, return not found.
	if _, isCredentials := secret.Labels[kargoapi.LabelKeyCredentialType]; !isCredentials {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"secret %s/%s exists, but is not labeled with %s",
				secret.Namespace,
				secret.Name,
				kargoapi.LabelKeyCredentialType,
			),
		)
	}

	if err := s.client.Delete(ctx, &secret); err != nil {
		return nil, fmt.Errorf("delete secret: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteRepoCredentialsResponse{}), nil
}
