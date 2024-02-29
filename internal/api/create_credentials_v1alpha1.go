package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type credentials struct {
	project        string
	name           string
	credType       string
	repoURL        string
	repoURLPattern string
	username       string
	password       string
}

func (s *server) CreateCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.CreateCredentialsRequest],
) (*connect.Response[svcv1alpha1.CreateCredentialsResponse], error) {
	creds := credentials{
		project:        req.Msg.GetProject(),
		name:           req.Msg.GetName(),
		credType:       req.Msg.GetType(),
		repoURL:        req.Msg.GetRepoUrl(),
		repoURLPattern: req.Msg.GetRepoUrlPattern(),
		username:       req.Msg.GetUsername(),
		password:       req.Msg.GetPassword(),
	}

	if err := s.validateCredentials(creds); err != nil {
		return nil, err
	}

	if err := s.client.Create(ctx, credentialsToSecret(creds)); err != nil {
		return nil, errors.Wrap(err, "create secret")
	}

	return connect.NewResponse(&svcv1alpha1.CreateCredentialsResponse{}), nil
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
	if creds.repoURL == "" && creds.repoURLPattern == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("at least one of repoURL or repoURLPattern should not be empty"),
		)
	}
	if err := validateFieldNotEmpty("username", creds.username); err != nil {
		return err
	}
	return validateFieldNotEmpty("password", creds.password)
}

func credentialsToSecret(creds credentials) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: creds.project,
			Name:      creds.name,
			Labels: map[string]string{
				kargoapi.CredentialTypeLabelKey: creds.credType,
			},
		},
		Data: map[string][]byte{
			"repoURL":        []byte(creds.repoURL),
			"repoURLPattern": []byte(creds.repoURLPattern),
			"username":       []byte(creds.username),
			"password":       []byte(creds.password),
		},
	}
}
