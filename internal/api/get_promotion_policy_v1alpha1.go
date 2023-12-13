package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetPromotionPolicy(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetPromotionPolicyRequest],
) (*connect.Response[svcv1alpha1.GetPromotionPolicyResponse], error) {
	if err := validateProjectAndStageNonEmpty(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}

	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var policy kargoapi.PromotionPolicy
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}, &policy); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "get promotion policy"))
	}
	return connect.NewResponse(&svcv1alpha1.GetPromotionPolicyResponse{
		PromotionPolicy: typesv1alpha1.ToPromotionPolicyProto(policy),
	}), nil
}
