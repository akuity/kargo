package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/internal/credentials"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type specificCredentials struct {
	project        string
	name           string
	credType       string
	repoURL        string
	repoURLIsRegex bool
	username       string
	password       string
	description    string
}

type genericCredentials struct {
	project     string
	name        string
	description string
	data        map[string]string
}

func (s *server) CreateCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateCredentialsRequest],
) (*connect.Response[svcv1alpha1.CreateCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	credType := req.Msg.GetType()

	if err := validateFieldNotEmpty("type", credType); err != nil {
		return nil, err
	}

	switch credType {
	case kargoapi.CredentialTypeLabelValueGit,
		kargoapi.CredentialTypeLabelValueHelm,
		kargoapi.CredentialTypeLabelValueImage:
		creds := specificCredentials{
			project:        req.Msg.GetProject(),
			name:           req.Msg.GetName(),
			description:    req.Msg.GetDescription(),
			credType:       req.Msg.GetType(),
			repoURL:        req.Msg.GetRepoUrl(),
			repoURLIsRegex: req.Msg.GetRepoUrlIsRegex(),
			username:       req.Msg.GetUsername(),
			password:       req.Msg.GetPassword(),
		}

		kubernetesSecret, err := s.createSpecificCredentials(ctx, creds)

		if err != nil {
			return nil, err
		}

		return connect.NewResponse(
			&svcv1alpha1.CreateCredentialsResponse{
				Credentials: kubernetesSecret,
			},
		), nil
	case kargoapi.CredentialTypeLabelValueGeneric:
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

		return connect.NewResponse(&svcv1alpha1.CreateCredentialsResponse{
			Credentials: kubernetesSecret,
		}), nil
	}

	return nil, connect.NewError(
		connect.CodeInvalidArgument,
		errors.New("type should be one of git, helm, image or generic"),
	)
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

// creates credentials used for known purpose; specifically external subscriptions
// to private git repo or helm OCI or docker image
func (s *server) createSpecificCredentials(ctx context.Context, creds specificCredentials) (*corev1.Secret, error) {
	if err := s.validateCredentials(creds); err != nil {
		return nil, err
	}

	secret := credentialsToSecret(creds)

	if err := s.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}

	return sanitizeCredentialSecret(*secret), nil
}

func (s *server) validateCredentials(creds specificCredentials) error {
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

func credentialsToSecret(creds specificCredentials) *corev1.Secret {
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
