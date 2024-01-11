package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func (s *server) ListPromotions(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListPromotionsRequest],
) (*connect.Response[svcv1alpha1.ListPromotionsResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}

	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var list kargoapi.PromotionList
	opts := []client.ListOption{
		client.InNamespace(req.Msg.GetProject()),
	}
	if req.Msg.GetStage() != "" {
		opts = append(opts,
			client.MatchingFields{kubeclient.PromotionsByStageIndexField: req.Msg.GetStage()},
		)
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, errors.Wrap(err, "list promotions")
	}
	promotions := make([]*v1alpha1.Promotion, len(list.Items))
	for idx, promotion := range list.Items {
		promotions[idx] = typesv1alpha1.ToPromotionProto(promotion)
	}
	return connect.NewResponse(&svcv1alpha1.ListPromotionsResponse{
		Promotions: promotions,
	}), nil
}
