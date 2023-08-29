package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargov1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) DeletePromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeletePromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.DeletePromotionPolicyResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
	}
	if err := validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var policy kargov1alpha1.PromotionPolicy
	key := client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}
	if err := s.client.Get(ctx, key, &policy); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("promotion policy %q not found", key.String()))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.client.Delete(ctx, &policy); err != nil && !kubeerr.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&svcv1alpha1.DeletePromotionPolicyResponse{}), nil
}
