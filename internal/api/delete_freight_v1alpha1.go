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
	if err := validateProjectAndFreightName(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
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
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "delete freight"))
	}
	return connect.NewResponse(&svcv1alpha1.DeleteFreightResponse{}), nil
}
