package api

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sort"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListAnalysisTemplates(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListAnalysisTemplatesRequest],
) (*connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("Argo Rollouts integration is not enabled"),
		)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list rollouts.AnalysisTemplateList
	opts := []client.ListOption{
		client.InNamespace(project),
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list analysistemplates: %w", err)
	}

	slices.SortFunc(list.Items, func (a, b rollouts.AnalysisTemplate) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		} else {
			return 0
		}
	})

	ats := make([]*rollouts.AnalysisTemplate, len(list.Items))
	for idx := range list.Items {
		ats[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListAnalysisTemplatesResponse{
		AnalysisTemplates: ats,
	}), nil
}
