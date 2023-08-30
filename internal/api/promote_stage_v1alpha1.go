package api

import (
	"context"

	"connectrpc.com/connect"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/kargo"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) PromoteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteStageRequest],
) (*connect.Response[svcv1alpha1.PromoteStageResponse], error) {
	if err := validateProjectAndStageNonEmpty(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}
	stage, err := getStage(ctx, s.client, req.Msg.GetProject(), req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	if err := validateFreightExists(req.Msg.GetState(), stage.Status.AvailableStates); err != nil {
		return nil, err
	}

	promotion := kargo.NewPromotion(*stage, req.Msg.GetState())
	if err := s.client.Create(ctx, &promotion); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&svcv1alpha1.PromoteStageResponse{
		Promotion: typesv1alpha1.ToPromotionProto(promotion),
	}), nil
}
