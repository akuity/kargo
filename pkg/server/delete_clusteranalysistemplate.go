package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// @id DeleteClusterAnalysisTemplate
// @Summary Delete a ClusterAnalysisTemplate
// @Description Delete a ClusterAnalysisTemplate resource.
// @Tags Verifications, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Param cluster-analysis-template path string true "ClusterAnalysisTemplate name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/shared/cluster-analysis-templates/{cluster-analysis-template} [delete]
func (s *server) deleteClusterAnalysisTemplate(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()

	name := c.Param("cluster-analysis-template")

	if err := s.client.Delete(ctx, &v1alpha1.ClusterAnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
