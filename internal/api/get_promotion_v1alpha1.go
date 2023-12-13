package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetPromotion(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetPromotionRequest],
) (*connect.Response[svcv1alpha1.GetPromotionResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var promotion kargoapi.Promotion
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}, &promotion); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				errors.Errorf("promotion %q not found", req.Msg.GetName()))
		}
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "get promotion"))
	}
	return connect.NewResponse(&svcv1alpha1.GetPromotionResponse{
		Promotion: v1alpha1.ToPromotionProto(promotion),
	}), nil
}
