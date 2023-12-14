package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdatePromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdatePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.UpdatePromotionPolicyResponse], error) {
	var policy kargoapi.PromotionPolicy
	switch {
	case req.Msg.GetYaml() != "":
		if err := yaml.Unmarshal([]byte(req.Msg.GetYaml()), &policy); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid yaml"))
		}
	case req.Msg.GetTyped() != nil:
		if req.Msg.GetTyped().GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if req.Msg.GetTyped().GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}
		policy = kargoapi.PromotionPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Msg.GetTyped().GetProject(),
				Name:      req.Msg.GetTyped().GetName(),
			},
			Stage:               req.Msg.GetTyped().GetStage(),
			EnableAutoPromotion: req.Msg.GetTyped().GetEnableAutoPromotion(),
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("promotion_policy should not be empty"))
	}

	if err := s.validateProject(ctx, policy.GetNamespace()); err != nil {
		return nil, err
	}
	var existingPolicy kargoapi.PromotionPolicy
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(&policy), &existingPolicy); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "get promotion policy"))
	}
	policy.SetResourceVersion(existingPolicy.GetResourceVersion())
	if err := s.client.Update(ctx, &policy); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "update promotion policy"))
	}
	return connect.NewResponse(&svcv1alpha1.UpdatePromotionPolicyResponse{
		PromotionPolicy: typesv1alpha1.ToPromotionPolicyProto(policy),
	}), nil
}
