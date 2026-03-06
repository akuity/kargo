package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

func (s *server) ListClusterAnalysisTemplates(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListClusterAnalysisTemplatesRequest],
) (*connect.Response[svcv1alpha1.ListClusterAnalysisTemplatesResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		// nolint:staticcheck
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

// nolint: lll
// @id ListClusterAnalysisTemplates
// @Summary List ClusterAnalysisTemplates
// @Description List ClusterAnalysisTemplate resources. Returns a
// @Description ClusterAnalysisTemplateList resource.
// @Tags Verifications, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "ClusterAnalysisTemplateList custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.ClusterAnalysisTemplateList)"
// @Router /v1beta1/shared/cluster-analysis-templates [get]
func (s *server) listClusterAnalysisTemplates(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()

	list := &rollouts.ClusterAnalysisTemplateList{}
	if err := s.client.List(ctx, list); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs rollouts.ClusterAnalysisTemplate) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}
