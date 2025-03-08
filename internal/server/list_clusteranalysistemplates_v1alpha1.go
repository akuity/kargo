package server

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListClusterAnalysisTemplates(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListClusterAnalysisTemplatesRequest],
) (*connect.Response[svcv1alpha1.ListClusterAnalysisTemplatesResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("Argo Rollouts integration is not enabled"),
		)
	}

	var list rollouts.ClusterAnalysisTemplateList
	if err := s.client.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("list clusteranalysistemplates: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs rollouts.ClusterAnalysisTemplate) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	cats := make([]*rollouts.ClusterAnalysisTemplate, len(list.Items))
	for idx := range list.Items {
		cats[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListClusterAnalysisTemplatesResponse{
		ClusterAnalysisTemplates: cats,
	}), nil
}
