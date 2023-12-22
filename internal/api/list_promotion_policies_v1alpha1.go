package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func (s *server) ListPromotionPolicies(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListPromotionPoliciesRequest],
) (*connect.Response[svcv1alpha1.ListPromotionPoliciesResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var list kargoapi.PromotionPolicyList
	if err := s.client.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
		return nil, errors.Wrap(err, "list promotion policies")
	}
	policies := make([]*v1alpha1.PromotionPolicy, len(list.Items))
	for idx, policy := range list.Items {
		policies[idx] = typesv1alpha1.ToPromotionPolicyProto(policy)
	}
	return connect.NewResponse(&svcv1alpha1.ListPromotionPoliciesResponse{
		PromotionPolicies: policies,
	}), nil
}
