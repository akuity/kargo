package server

import (
	"context"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/indexer"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
		opts = append(opts, client.MatchingFields{indexer.PromotionsByStageField: stage})
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list promotions: %w", err)
	}

	slices.SortFunc(list.Items, api.ComparePromotionByPhaseAndCreationTime)

	promotions := make([]*kargoapi.Promotion, len(list.Items))
	for idx := range list.Items {
		promotions[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListPromotionsResponse{
		Promotions: promotions,
	}), nil
}
