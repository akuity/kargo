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

type genericCredentials struct {
	systemLevel bool
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateGenericCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateGenericCredentialsRequest],
) (*connect.Response[svcv1alpha1.CreateGenericCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	genCreds := genericCredentials{
		systemLevel: req.Msg.SystemLevel,
		project:     req.Msg.Project,
		name:        req.Msg.Name,
		data:        req.Msg.Data,
		description: req.Msg.Description,
	}

	if err := s.validateGenericCredentials(ctx, genCreds); err != nil {
		return nil, err
	}

	secret := s.genericCredentialsToK8sSecret(genCreds)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateGenericCredentialsResponse{
			Credentials: svcv1alpha1.FromK8sSecret(sanitizeGenericCredentials(*secret)),
		},
	), nil
}

func (s *server) validateGenericCredentials(
	ctx context.Context,
	genCreds genericCredentials,
) error {
	if !genCreds.systemLevel && genCreds.project != "" {
		if err := s.validateProjectExists(ctx, genCreds.project); err != nil {
			return err
		}
	}

	if err := validateFieldNotEmpty("name", genCreds.name); err != nil {
		return err
	}

	if len(genCreds.data) == 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			errors.New("cannot create empty secret"))
	}

	return nil
}

func (s *server) genericCredentialsToK8sSecret(genCreds genericCredentials) *corev1.Secret {
	secretsData := map[string][]byte{}

	for key, value := range genCreds.data {
		secretsData[key] = []byte(value)
	}

	var namespace string
	if genCreds.systemLevel {
		namespace = s.cfg.SystemResourcesNamespace
	} else {
		namespace = genCreds.project
		if namespace == "" {
			namespace = s.cfg.SharedResourcesNamespace
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      genCreds.name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: secretsData,
	}

	if genCreds.description != "" {
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: genCreds.description,
		}
	}

	return secret
}
