package api

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListCredentialsRequest],
) (*connect.Response[svcv1alpha1.ListCredentialsResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var secretsList corev1.SecretList
	if err := s.client.List(
		ctx,
		&secretsList,
		client.InNamespace(req.Msg.GetProject()),
		client.HasLabels{kargoapi.CredentialTypeLabelKey},
	); err != nil {
		return nil, errors.Wrap(err, "list secrets")
	}

	sort.Slice(secretsList.Items, func(i, j int) bool {
		return secretsList.Items[i].Name < secretsList.Items[j].Name
	})

	secrets := make([]*corev1.Secret, len(secretsList.Items))
	for i, secret := range secretsList.Items {
		secrets[i] = sanitizeCredentialSecret(secret)
	}

	return connect.NewResponse(&svcv1alpha1.ListCredentialsResponse{
		Credentials: secrets,
	}), nil
}
