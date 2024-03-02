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
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	stageName := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", stageName); err != nil {
		return nil, err
	}

	freightName := req.Msg.GetFreight()
	if err := validateFieldNotEmpty("freight", freightName); err != nil {
		return nil, err
	}

	if err := s.validateProjectExistsFn(ctx, project); err != nil {
		return nil, err
	}

	stage, err := s.getStageFn(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: project,
			Name:      stageName,
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
				stageName,
				project,
			),
		)
	}

	freight, err := s.getFreightFn(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: project,
			Name:      freightName,
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
				freightName,
				project,
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
				freightName,
				stageName,
			),
		)
	}

	promotion := kargo.NewPromotion(*stage, freightName)
	if err := s.createPromotionFn(ctx, &promotion); err != nil {
		return nil, errors.Wrap(err, "create promotion")
	}
	return connect.NewResponse(&svcv1alpha1.PromoteStageResponse{
		Promotion: typesv1alpha1.ToPromotionProto(promotion),
	}), nil
}
