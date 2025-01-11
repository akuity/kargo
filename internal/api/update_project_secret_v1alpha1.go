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

func (s *server) UpdateProjectSecret(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateProjectSecretRequest],
) (*connect.Response[svcv1alpha1.UpdateProjectSecretResponse], error) {
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
		return nil, fmt.Errorf("get secret: %w", err)
	}

	// If this isn't labeled as a project secret, return not found.
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

	projectSecretUpdate := projectSecret{
		data:        req.Msg.GetData(),
		description: req.Msg.GetDescription(),
	}

	if len(projectSecretUpdate.data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot create empty secret"))
	}

	applyProjectSecretUpdateToK8sSecret(&secret, projectSecretUpdate)

	if err := s.client.Update(ctx, &secret); err != nil {
		return nil, fmt.Errorf("update secret: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.UpdateProjectSecretResponse{
			Secret: sanitizeProjectSecret(secret),
		},
	), nil
}

func applyProjectSecretUpdateToK8sSecret(secret *corev1.Secret, projectSecretUpdate projectSecret) {
	if projectSecretUpdate.description != "" {
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string, 1)
		}
		secret.Annotations[kargoapi.AnnotationKeyDescription] = projectSecretUpdate.description
	} else {
		delete(secret.Annotations, kargoapi.AnnotationKeyDescription)
	}

	// delete the keys that exist in secret but not in the update
	for key := range secret.Data {
		if _, exist := projectSecretUpdate.data[key]; !exist {
			delete(secret.Data, key)
		}
	}

	// upsert
	if secret.Data == nil {
		secret.Data = make(map[string][]byte, len(projectSecretUpdate.data))
	}
	for key, value := range projectSecretUpdate.data {
		if value != "" {
			secret.Data[key] = []byte(value)
		}
	}
}
