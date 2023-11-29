package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeleteFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteFreightRequest],
) (*connect.Response[svcv1alpha1.DeleteFreightResponse], error) {
	if err := validateProjectAndFreightName(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}
	var freight kargoapi.Freight
	if err := s.client.Get(
		ctx,
		client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		},
		&freight,
	); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, connect.NewError(
				connect.CodeNotFound,
				errors.Errorf("freight %q not found", req.Msg.GetName()),
			)
		}
		return nil, errors.Wrap(err, "get freight")
	}
	if err := s.client.Delete(ctx, &freight); err != nil {
		return nil, errors.Wrap(err, "delete freight")
	}
	return connect.NewResponse(&svcv1alpha1.DeleteFreightResponse{}), nil
}
