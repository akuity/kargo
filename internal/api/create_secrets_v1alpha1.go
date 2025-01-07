package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type genericCredentials struct {
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateSecrets(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateSecretsRequest],
) (*connect.Response[svcv1alpha1.CreateSecretsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	creds := genericCredentials{
		project:     req.Msg.GetProject(),
		name:        req.Msg.GetName(),
		data:        req.Msg.GetData(),
		description: req.Msg.GetDescription(),
	}

	kubernetesSecret, err := s.createGenericCredentials(ctx, creds)

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&svcv1alpha1.CreateSecretsResponse{
		Secret: kubernetesSecret,
	}), nil
}

func (s *server) genericCredentialsToSecret(creds genericCredentials) *corev1.Secret {
	secretsData := map[string][]byte{}

	for key, value := range creds.data {
		secretsData[key] = []byte(value)
	}

	kubernetesSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: creds.project,
			Name:      creds.name,
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: kargoapi.CredentialTypeLabelValueGeneric,
			},
		},
		Data: secretsData,
	}

	if creds.description != "" {
		kubernetesSecret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: creds.description,
		}
	}

	return kubernetesSecret
}

func (s *server) createGenericCredentials(ctx context.Context, creds genericCredentials) (*corev1.Secret, error) {
	if err := s.validateGenericCredentials(creds); err != nil {
		return nil, err
	}

	kubernetesSecret := s.genericCredentialsToSecret(creds)

	if err := s.client.Create(ctx, kubernetesSecret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	// redact
	// TODO: current function does not redact some keys, create new function
	redactedSecret := sanitizeCredentialSecret(*kubernetesSecret)

	return redactedSecret, nil
}

func (s *server) validateGenericCredentials(creds genericCredentials) error {
	if err := validateFieldNotEmpty("project", creds.project); err != nil {
		return err
	}

	if err := validateFieldNotEmpty("name", creds.name); err != nil {
		return err
	}

	if len(creds.data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
	}

	return nil
}
