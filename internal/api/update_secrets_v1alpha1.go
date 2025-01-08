package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdateSecrets(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateSecretsRequest],
) (*connect.Response[svcv1alpha1.UpdateSecretsResponse], error) {
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

	secret := corev1.Secret{}
	if err := s.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: project,
			Name:      name,
		},
		&secret,
	); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("secret %s/%s not found", project, name))
	}

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

	genericCredsUpdate := genericCredentials{
		data:        req.Msg.GetData(),
		description: req.Msg.GetDescription(),
	}

	if len(genericCredsUpdate.data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
	}

	applyGenericCredentialsUpdateToSecret(&secret, genericCredsUpdate)

	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateSecretsResponse{
			Secret: sanitizeSecret(secret, []string{}),
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
