package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// nolint: lll
// @id ListClusterAnalysisTemplates
// @Summary List ClusterAnalysisTemplates
// @Description List ClusterAnalysisTemplate resources. Returns a
// @Description ClusterAnalysisTemplateList resource.
// @Tags Verifications, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Success 200 {object} rollouts.ClusterAnalysisTemplateList "ClusterAnalysisTemplateList custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.ClusterAnalysisTemplateList)"
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
