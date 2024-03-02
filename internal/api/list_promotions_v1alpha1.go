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
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	stage := req.Msg.GetStage()

	var list kargoapi.PromotionList
	opts := []client.ListOption{
		client.InNamespace(project),
	}
	if stage != "" {
		opts = append(opts,
			client.MatchingFields{kubeclient.PromotionsByStageIndexField: stage},
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
