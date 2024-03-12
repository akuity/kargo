package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type credentialsUpdate struct {
	project        string
	name           string
	credType       string
	repoURL        string
	repoURLPattern string
	username       string
	password       string
}

func (s *server) UpdateCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateCredentialsRequest],
) (*connect.Response[svcv1alpha1.UpdateCredentialsResponse], error) {
	credsUpdate := credentialsUpdate{
		project:        req.Msg.GetProject(),
		name:           req.Msg.GetName(),
		credType:       req.Msg.GetType(),
		repoURL:        req.Msg.GetRepoUrl(),
		repoURLPattern: req.Msg.GetRepoUrlPattern(),
		username:       req.Msg.GetUsername(),
		password:       req.Msg.GetPassword(),
	}

	if err := validateFieldNotEmpty("project", credsUpdate.project); err != nil {
		return nil, err
	}

	if err := validateFieldNotEmpty("name", credsUpdate.name); err != nil {
		return nil, err
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: credsUpdate.project,
			Name:      credsUpdate.name,
		},
		&secret,
	); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, errors.Wrap(err, "get secret")
	}

	// If this isn't labeled as repository credentials, return not found.
	var isCredentials bool
	if secret.Labels != nil {
		_, isCredentials = secret.Labels[kargoapi.CredentialTypeLabelKey]
	}
	if !isCredentials {
		return nil, connect.NewError(
			connect.CodeNotFound,
			errors.Errorf(
				"secret %q exists, but is not labeled with %q",
				secret.Name,
				kargoapi.CredentialTypeLabelKey,
			),
		)
	}

	applyCredentialsUpdateToSecret(&secret, credsUpdate)

	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, errors.Wrap(err, "update secret")
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateCredentialsResponse{
			Credentials: sanitizeCredentialSecret(secret),
		},
	), nil
}

func applyCredentialsUpdateToSecret(
	secret *corev1.Secret,
	credsUpdate credentialsUpdate,
) {
	if credsUpdate.credType != "" {
		secret.Labels[kargoapi.CredentialTypeLabelKey] = credsUpdate.credType
	}
	if credsUpdate.repoURL != "" {
		secret.Data["repoURL"] = []byte(credsUpdate.repoURL)
		delete(secret.Data, "repoURLPattern")
	}
	if credsUpdate.repoURLPattern != "" {
		secret.Data["repoURLPattern"] = []byte(credsUpdate.repoURLPattern)
		delete(secret.Data, "repoURL")
	}
	if credsUpdate.username != "" {
		secret.Data["username"] = []byte(credsUpdate.username)
	}
	if credsUpdate.password != "" {
		secret.Data["password"] = []byte(credsUpdate.password)
	}
}
