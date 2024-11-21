package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libCreds "github.com/akuity/kargo/internal/credentials"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type credentialsUpdate struct {
	project        string
	name           string
	description    string
	credType       string
	repoURL        string
	repoURLISRegex bool
	username       string
	password       string
}

func (s *server) UpdateCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateCredentialsRequest],
) (*connect.Response[svcv1alpha1.UpdateCredentialsResponse], error) {
	// Check if secret management is enabled
	if !s.cfg.SecretManagementEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errSecretManagementDisabled)
	}

	credType := req.Msg.GetType()
	project := req.Msg.GetProject()
	name := req.Msg.GetName()

	if err := validateFieldNotEmpty("type", credType); err != nil {
		return nil, err
	}

	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		&secret,
	); err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}

	// If this isn't labeled as repository credentials, return not found.
	var isCredentials bool
	if secret.Labels != nil {
		_, isCredentials = secret.Labels[kargoapi.CredentialTypeLabelKey]
	}
	if !isCredentials {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"secret %q exists, but is not labeled with %q",
				secret.Name,
				kargoapi.CredentialTypeLabelKey,
			),
		)
	}

	switch credType {
	case kargoapi.CredentialTypeLabelValueGit:
	case kargoapi.CredentialTypeLabelValueHelm:
	case kargoapi.CredentialTypeLabelValueImage:

		credsUpdate := credentialsUpdate{
			project:        req.Msg.GetProject(),
			name:           req.Msg.GetName(),
			description:    req.Msg.GetDescription(),
			credType:       req.Msg.GetType(),
			repoURL:        req.Msg.GetRepoUrl(),
			repoURLISRegex: req.Msg.GetRepoUrlIsRegex(),
			username:       req.Msg.GetUsername(),
			password:       req.Msg.GetPassword(),
		}

		applyCredentialsUpdateToSecret(&secret, credsUpdate)

	case kargoapi.CredentialTypeLabelValueGeneric:
		genericCredsUpdate := genericCredentials{
			data:        req.Msg.GetData(),
			description: req.Msg.GetDescription(),
		}

		if len(genericCredsUpdate.data) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
		}

		applyGenericCredentialsUpdateToSecret(&secret, genericCredsUpdate)
	default:
		return nil, connect.NewError(
			connect.CodeInvalidArgument, errors.New("type should be one of git, helm, image or generic"),
		)
	}

	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateCredentialsResponse{
			Credentials: sanitizeCredentialSecret(secret),
		},
	), nil
}

func applyGenericCredentialsUpdateToSecret(secret *corev1.Secret, genericCredsUpdate genericCredentials) {
	if genericCredsUpdate.description != "" {
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string, 1)
		}
		secret.Annotations[kargoapi.AnnotationKeyDescription] = genericCredsUpdate.description
	} else {
		delete(secret.Annotations, kargoapi.AnnotationKeyDescription)
	}

	// delete the keys that exist in secret but not in updater
	for key := range secret.Data {
		_, exist := genericCredsUpdate.data[key]

		if !exist {
			delete(secret.Data, key)
		}
	}

	// upsert
	for key, value := range genericCredsUpdate.data {
		_, existInSecret := secret.Data[key]

		if !existInSecret || (existInSecret && value != "") {
			secret.Data[key] = []byte(value)
			continue
		}
	}
}

func applyCredentialsUpdateToSecret(
	secret *corev1.Secret,
	credsUpdate credentialsUpdate,
) {
	if credsUpdate.description != "" {
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string, 1)
		}
		secret.Annotations[kargoapi.AnnotationKeyDescription] = credsUpdate.description
	} else {
		delete(secret.Annotations, kargoapi.AnnotationKeyDescription)
	}

	if credsUpdate.credType != "" {
		secret.Labels[kargoapi.CredentialTypeLabelKey] = credsUpdate.credType
	}
	if credsUpdate.repoURL != "" {
		secret.Data[libCreds.FieldRepoURL] = []byte(credsUpdate.repoURL)
		if credsUpdate.repoURLISRegex {
			secret.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
		} else {
			delete(secret.Data, libCreds.FieldRepoURLIsRegex)
		}
	}
	if credsUpdate.username != "" {
		secret.Data["username"] = []byte(credsUpdate.username)
	}
	if credsUpdate.password != "" {
		secret.Data["password"] = []byte(credsUpdate.password)
	}
}
