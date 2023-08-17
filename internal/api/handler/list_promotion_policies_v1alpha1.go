package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

type ListPromotionPoliciesV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.ListPromotionPoliciesRequest],
) (*connect.Response[svcv1alpha1.ListPromotionPoliciesResponse], error)

func ListPromotionPoliciesV1Alpha1(
	kc client.Client,
) ListPromotionPoliciesV1Alpha1Func {
	validateProject := newProjectValidator(kc)
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.ListPromotionPoliciesRequest],
	) (*connect.Response[svcv1alpha1.ListPromotionPoliciesResponse], error) {
		if req.Msg.GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if err := validateProject(ctx, req.Msg.GetProject()); err != nil {
			return nil, err
		}

		var list kubev1alpha1.PromotionPolicyList
		if err := kc.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		policies := make([]*v1alpha1.PromotionPolicy, len(list.Items))
		for idx, policy := range list.Items {
			policies[idx] = typesv1alpha1.ToPromotionPolicyProto(policy)
		}
		return connect.NewResponse(&svcv1alpha1.ListPromotionPoliciesResponse{
			PromotionPolicies: policies,
		}), nil
	}
}
