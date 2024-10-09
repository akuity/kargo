package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kargo"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

// PromoteDownstream creates Promotion resources to transition all Stages
// immediately downstream from the specified Stage into the state represented by
// the specified Freight.
func (s *server) PromoteDownstream(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.PromoteDownstreamRequest],
) (*connect.Response[svcv1alpha1.PromoteDownstreamResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	stageName := req.Msg.GetStage()
	if err := validateFieldNotEmpty("stage", stageName); err != nil {
		return nil, err
	}

	freightName := req.Msg.GetFreight()
	freightAlias := req.Msg.GetFreightAlias()
	if (freightName == "" && freightAlias == "") || (freightName != "" && freightAlias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of freightName or freightAlias should not be empty"),
		)
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
		return nil, fmt.Errorf("get stage: %w", err)
	}
	if stage == nil {
		return nil, connect.NewError(
			connect.CodeNotFound,
			fmt.Errorf(
				"Stage %q not found in namespace %q",
				stageName,
				project,
			),
		)
	}

	// Get the specified Freight, but only if it is verified in this Stage.
	// Merely being approved FOR this Stage is not enough. If Freight is only
	// approved FOR this Stage, that is because someone manually did that. This
	// does not speak to its suitability for promotion downstream. If a user
	// desires to promote Freight downstream that is not verified in this
	// Stage, then they should approve the Freight for the downstream Stage(s).
	// Expect a nil if the specified Freight is not found or doesn't meet these
	// conditions. Errors are indicative only of internal problems.
	freight, err := s.getFreightByNameOrAliasFn(
		ctx,
		s.client,
		project,
		freightName,
		freightAlias,
	)
	if err != nil {
		return nil, fmt.Errorf("get freight: %w", err)
	}
	if freight == nil {
		if freightName != "" {
			err = fmt.Errorf("freight %q not found in namespace %q", freightName, project)
		} else {
			err = fmt.Errorf("freight with alias %q not found in namespace %q", freightAlias, project)
		}
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if !s.isFreightAvailableFn(
		freight,
		"",                  // approved for not considered
		[]string{stageName}, // verified in
	) {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"Freight %q is not available to Stage %q",
				freightName,
				stageName,
			),
		)
	}

	downstreams, err := s.findDownstreamStagesFn(ctx, stage)
	if err != nil {
		return nil, fmt.Errorf("find downstream stages: %w", err)
	}
	if len(downstreams) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("stage %q has no downstream stages", stageName))
	}

	for _, downstream := range downstreams {
		if err := s.authorizeFn(
			ctx,
			"promote",
			kargoapi.GroupVersion.WithResource("stages"),
			"",
			types.NamespacedName{
				Namespace: downstream.Namespace,
				Name:      downstream.Name,
			},
		); err != nil {
			return nil, err
		}
	}

	promoteErrs := make([]error, 0, len(downstreams))
	createdPromos := make([]*kargoapi.Promotion, 0, len(downstreams))
	for _, downstream := range downstreams {
		newPromo := kargo.NewPromotion(ctx, downstream, freight.Name)
		if downstream.Spec.PromotionTemplate != nil &&
			len(downstream.Spec.PromotionTemplate.Spec.Steps) == 0 {
			// Avoid creating a Promotion if the downstream Stage has no promotion
			// steps and is therefore a "control flow" Stage.
			continue
		}
		if err := s.createPromotionFn(ctx, &newPromo); err != nil {
			promoteErrs = append(promoteErrs, err)
			continue
		}
		s.recordPromotionCreatedEvent(ctx, &newPromo, freight)
		createdPromos = append(createdPromos, &newPromo)
	}

	res := connect.NewResponse(&svcv1alpha1.PromoteDownstreamResponse{
		Promotions: createdPromos,
	})

	if len(promoteErrs) > 0 {
		return res, connect.NewError(connect.CodeInternal, errors.Join(promoteErrs...))
	}

	return res, nil
}

// findDownstreamStages returns a list of Stages that are immediately downstream
// from the given Stage.
// TODO: this could be powered by an index.
func (s *server) findDownstreamStages(ctx context.Context, stage *kargoapi.Stage) ([]kargoapi.Stage, error) {
	var allStages kargoapi.StageList
	if err := s.client.List(ctx, &allStages, client.InNamespace(stage.Namespace)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var downstreams []kargoapi.Stage
	for _, s := range allStages.Items {
		for _, req := range s.Spec.RequestedFreight {
			for _, upstream := range req.Sources.Stages {
				if upstream == stage.Name {
					downstreams = append(downstreams, s)
				}
			}
		}
	}
	return downstreams, nil
}
