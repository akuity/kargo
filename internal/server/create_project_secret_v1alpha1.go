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

type projectSecret struct {
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateProjectSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateProjectSecretRequest],
) (*connect.Response[svcv1alpha1.CreateProjectSecretResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	projSecret := projectSecret{
		project:     req.Msg.GetProject(),
		name:        req.Msg.GetName(),
		data:        req.Msg.GetData(),
		description: req.Msg.GetDescription(),
	}

	if err := s.validateProjectSecret(projSecret); err != nil {
		return nil, err
	}

	secret := s.projectSecretToK8sSecret(projSecret)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateProjectSecretResponse{
			Secret: sanitizeProjectSecret(*secret),
		},
	), nil
}

func (s *server) validateProjectSecret(projSecret projectSecret) error {
	if err := validateFieldNotEmpty("project", projSecret.project); err != nil {
		return err
	}

	if err := validateFieldNotEmpty("name", projSecret.name); err != nil {
		return err
	}

	if len(projSecret.data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			errors.New("cannot create empty secret"))
	}

	return nil
}

func (s *server) projectSecretToK8sSecret(projSecret projectSecret) *corev1.Secret {
	secretsData := map[string][]byte{}

	for key, value := range projSecret.data {
		secretsData[key] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: projSecret.project,
			Name:      projSecret.name,
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGeneric,
			},
		},
		Data: secretsData,
	}

	if projSecret.description != "" {
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: projSecret.description,
		}
	}

	return secret
}
