package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteFreightRequest],
) (*connect.Response[svcv1alpha1.DeleteFreightResponse], error) {
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

	if err := s.client.Delete(ctx, &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		},
	}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, connect.NewError(
				connect.CodeNotFound,
				errors.Errorf("freight %q not found", req.Msg.GetName()),
			)
		}
		return nil, errors.Wrap(err, "delete freight")
	}
	return connect.NewResponse(&svcv1alpha1.DeleteFreightResponse{}), nil
}
