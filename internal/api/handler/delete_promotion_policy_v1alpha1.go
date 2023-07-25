package handler

import (
	"context"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type DeletePromotionPolicyV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.DeletePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.DeletePromotionPolicyResponse], error)

func DeletePromotionPolicyV1Alpha1(
	kc client.Client,
) DeletePromotionPolicyV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.DeletePromotionPolicyRequest],
	) (*connect.Response[svcv1alpha1.DeletePromotionPolicyResponse], error) {
		if req.Msg.GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if req.Msg.GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}

		var policy kubev1alpha1.PromotionPolicy
		key := client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		}
		if err := kc.Get(ctx, key, &policy); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("promotion policy %q not found", key.String()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := kc.Delete(ctx, &policy); err != nil && !kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.DeletePromotionPolicyResponse{}), nil
	}
}
