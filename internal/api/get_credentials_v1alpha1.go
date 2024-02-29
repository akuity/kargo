package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetCredentialsRequest],
) (*connect.Response[svcv1alpha1.GetCredentialsResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
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
		return nil, errors.Wrapf(err, "get secret %s", name)
	}

	secret = redactPassword(secret)

	return connect.NewResponse(&svcv1alpha1.GetCredentialsResponse{
		Credentials: typesv1alpha1.ToSecretProto(&secret),
	}), nil
}

func redactPassword(secret corev1.Secret) corev1.Secret {
	secret.StringData = make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		if k == "password" {
			secret.StringData[k] = "*** REDACTED ***"
		} else {
			secret.StringData[k] = string(v)
		}
	}
	secret.Data = nil
	return secret
}
