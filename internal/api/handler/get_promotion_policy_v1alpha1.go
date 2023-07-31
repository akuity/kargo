package handler

import (
	"context"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type GetPromotionPolicyV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.GetPromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.GetPromotionPolicyResponse], error)

func GetPromotionPolicyV1Alpha1(
	kc client.Client,
) GetPromotionPolicyV1Alpha1Func {
	validateProject := newProjectValidator(kc)
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.GetPromotionPolicyRequest],
	) (*connect.Response[svcv1alpha1.GetPromotionPolicyResponse], error) {
		if req.Msg.GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if req.Msg.GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}
		if err := validateProject(ctx, req.Msg.GetProject()); err != nil {
			return nil, err
		}

		var policy kubev1alpha1.PromotionPolicy
		if err := kc.Get(ctx, client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		}, &policy); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetPromotionPolicyResponse{
			PromotionPolicy: typesv1alpha1.ToPromotionPolicyProto(policy),
		}), nil
	}
}
