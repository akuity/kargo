package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) SetAutoPromotionForStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.SetAutoPromotionForStageRequest],
) (*connect.Response[svcv1alpha1.SetAutoPromotionForStageResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if req.Msg.GetStage() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("stage should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}
	project, err := kargoapi.GetProject(ctx, s.client, req.Msg.GetProject())
	if err != nil {
		return nil, errors.Wrap(err, "get project")
	}
	if project.Spec.PromotionPolicies == nil {
		project.Spec.PromotionPolicies = []kargoapi.PromotionPolicy{}
	}
	var set bool
	for i, policy := range project.Spec.PromotionPolicies {
		if policy.Stage == req.Msg.GetStage() {
			project.Spec.PromotionPolicies[i].AutoPromotionEnabled = req.Msg.GetEnable()
			set = true
			break
		}
	}
	if !set {
		project.Spec.PromotionPolicies = append(
			project.Spec.PromotionPolicies,
			kargoapi.PromotionPolicy{
				Stage:                req.Msg.GetStage(),
				AutoPromotionEnabled: req.Msg.GetEnable(),
			},
		)
	}
	if err := s.client.Update(ctx, project); err != nil {
		return nil, errors.Wrap(err, "update project")
	}
	return connect.NewResponse(&svcv1alpha1.SetAutoPromotionForStageResponse{}), nil
}
