package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteProjectSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteProjectSecretRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectSecretResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}
	if secret.Labels[kargoapi.ProjectSecretLabelKey] != kargoapi.LabelTrueValue {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"secret %s/%s exists, but is not labeled with %s=%s",
				secret.Namespace,
				secret.Name,
				kargoapi.ProjectSecretLabelKey,
				kargoapi.LabelTrueValue,
			),
		)
	}

	if err := s.client.Delete(ctx, secret); err != nil {
		return nil, fmt.Errorf("delete secret: %w", err)
	}

	return connect.NewResponse(&svcv1alpha1.DeleteProjectSecretResponse{}), nil
}
