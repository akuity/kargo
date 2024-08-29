package api

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListAnalysisTemplateConfigMaps(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListAnalysisTemplateConfigMapsRequest],
) (*connect.Response[svcv1alpha1.ListAnalysisTemplateConfigMapsResponse], error) {
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

	var list corev1.ConfigMapList
	opts := []client.ListOption{
		client.InNamespace(project),
		client.MatchingLabels{
			kargoapi.AnalysisEnvLabelKey: kargoapi.LabelTrueValue,
		},
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list ConfigMaps: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs corev1.ConfigMap) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	cms := make([]*corev1.ConfigMap, len(list.Items))
	for idx := range list.Items {
		cms[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListAnalysisTemplateConfigMapsResponse{
		ConfigMaps: cms,
	}), nil
}
