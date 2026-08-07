package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// nolint: lll
// @id GetClusterAnalysisTemplate
// @Summary Retrieve a ClusterAnalysisTemplate
// @Description Retrieve a ClusterAnalysisTemplate by name.
// @Tags Verifications, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Param cluster-analysis-template path string true "ClusterAnalysisTemplate name"
// @Produce json
// @Success 200 {object} rolloutsapi.ClusterAnalysisTemplate "ClusterAnalysisTemplate custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.ClusterAnalysisTemplate)"
// @Router /v1beta1/shared/cluster-analysis-templates/{cluster-analysis-template} [get]
func (s *server) getClusterAnalysisTemplate(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()
	name := c.Param("cluster-analysis-template")
	template := &rolloutsapi.ClusterAnalysisTemplate{}

	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name}, template,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, template)
}
