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
	libCreds "github.com/akuity/kargo/internal/credentials"
)

type credentials struct {
	project        string
	name           string
	credType       string
	repoURL        string
	repoURLIsRegex bool
	username       string
	password       string
	description    string
}

func (s *server) CreateCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateCredentialsRequest],
) (*connect.Response[svcv1alpha1.CreateCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	creds := credentials{
		project:        req.Msg.GetProject(),
		name:           req.Msg.GetName(),
		description:    req.Msg.GetDescription(),
		credType:       req.Msg.GetType(),
		repoURL:        req.Msg.GetRepoUrl(),
		repoURLIsRegex: req.Msg.GetRepoUrlIsRegex(),
		username:       req.Msg.GetUsername(),
		password:       req.Msg.GetPassword(),
	}

	if err := s.validateCredentials(creds); err != nil {
		return nil, err
	}

	secret := credentialsToK8sSecret(creds)
	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.CreateCredentialsResponse{
			Credentials: sanitizeCredentialSecret(*secret),
		},
	), nil
}

func (s *server) validateCredentials(creds credentials) error {
	if err := validateFieldNotEmpty("project", creds.project); err != nil {
		return err
	}
	if err := validateFieldNotEmpty("name", creds.name); err != nil {
		return err
	}
	if err := validateFieldNotEmpty("type", creds.credType); err != nil {
		return err
	}
	switch creds.credType {
	case kargoapi.CredentialTypeLabelValueGit,
		kargoapi.CredentialTypeLabelValueHelm,
		kargoapi.CredentialTypeLabelValueImage:
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("type should be one of git, helm, or image"),
		)
	}
	if creds.repoURL == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("repoURL should not be empty"),
		)
	}
	if err := validateFieldNotEmpty("username", creds.username); err != nil {
		return err
	}
	return validateFieldNotEmpty("password", creds.password)
}

func credentialsToK8sSecret(creds credentials) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: creds.project,
			Name:      creds.name,
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: creds.credType,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte(creds.repoURL),
			libCreds.FieldUsername: []byte(creds.username),
			libCreds.FieldPassword: []byte(creds.password),
		},
	}
	if creds.description != "" {
		s.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: creds.description,
		}
	}
	if creds.repoURLIsRegex {
		s.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
	}
	return s
}
