package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteCredentials(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteCredentialsRequest],
) (*connect.Response[svcv1alpha1.DeleteCredentialsResponse], error) {
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

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: project,
			Name:      name,
		},
	}
	if err := s.client.Delete(ctx, secret); err != nil {
		if !kubeerr.IsNotFound(err) {
			return nil, errors.Wrap(err, "delete secret")
		}
	}

	return connect.NewResponse(&svcv1alpha1.DeleteCredentialsResponse{}), nil
}
