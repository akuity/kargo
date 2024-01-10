package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/kargo"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

// PromoteStage creates a Promotion resource to transition a specified Stage
// into the state represented by the specified Freight.
func (s *server) PromoteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteStageRequest],
) (*connect.Response[svcv1alpha1.PromoteStageResponse], error) {
	if err := validateProjectAndStageNonEmpty(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err // This already returns a connect.Error
	}
	if err := s.validateProjectFn(ctx, req.Msg.GetProject()); err != nil {
		return nil, err // This already returns a connect.Error
	}
	stage, err := s.getStageFn(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "get stage")
	}
	if stage == nil {
		return nil, connect.NewError(
			connect.CodeNotFound,
			errors.Errorf(
				"Stage %q not found in namespace %q",
				req.Msg.GetName(),
				req.Msg.GetProject(),
			),
		)
	}

	freight, err := s.getFreightFn(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetFreight(),
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "get freight")
	}
	if freight == nil {
		return nil, connect.NewError(
			connect.CodeNotFound,
			errors.Errorf(
				"Freight %q not found in namespace %q",
				req.Msg.GetFreight(),
				req.Msg.GetProject(),
			),
		)
	}
	upstreamStages := make([]string, len(stage.Spec.Subscriptions.UpstreamStages))
	for i, upstreamStage := range stage.Spec.Subscriptions.UpstreamStages {
		upstreamStages[i] = upstreamStage.Name
	}
	if !s.isFreightAvailableFn(freight, stage.Name, upstreamStages) {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.Errorf(
				"Freight %q is not available to Stage %q",
				req.Msg.GetFreight(),
				req.Msg.GetName(),
			),
		)
	}

	promotion := kargo.NewPromotion(*stage, req.Msg.GetFreight())
	if err := s.createPromotionFn(ctx, &promotion); err != nil {
		return nil, errors.Wrap(err, "create promotion")
	}
	return connect.NewResponse(&svcv1alpha1.PromoteStageResponse{
		Promotion: typesv1alpha1.ToPromotionProto(promotion),
	}), nil
}
