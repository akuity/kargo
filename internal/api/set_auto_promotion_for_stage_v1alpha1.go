package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
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

	var policyList kargoapi.PromotionPolicyList
	if err := s.client.List(ctx, &policyList, client.InNamespace(req.Msg.GetProject()), client.MatchingFields{
		kubeclient.PromotionPoliciesByStageIndexField: req.Msg.GetStage(),
	}); err != nil {
		return nil, errors.Wrap(err, "list promotion policies")
	}

	// Since only one PromotionPolicy is allowed per stage,
	// create if not exists and update if exists.
	var policy kargoapi.PromotionPolicy
	if len(policyList.Items) > 0 {
		policy = policyList.Items[0]
		policy.EnableAutoPromotion = req.Msg.GetEnable()
		if err := s.client.Update(ctx, &policy); err != nil {
			return nil, errors.Wrap(err, "update promotion policy")
		}
	} else {
		policy = kargoapi.PromotionPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Msg.GetProject(),
				Name:      req.Msg.GetStage(),
			},
			Stage:               req.Msg.GetStage(),
			EnableAutoPromotion: req.Msg.GetEnable(),
		}
		if err := s.client.Create(ctx, &policy); err != nil {
			return nil, errors.Wrap(err, "create promotion policy")
		}
	}
	return connect.NewResponse(&svcv1alpha1.SetAutoPromotionForStageResponse{
		PromotionPolicy: typesv1alpha1.ToPromotionPolicyProto(policy),
	}), nil
}
