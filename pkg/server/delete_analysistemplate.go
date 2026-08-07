package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// @id DeleteAnalysisTemplate
// @Summary Delete an AnalysisTemplate
// @Description Delete an AnalysisTemplate resource from a project's namespace.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param analysis-template path string true "AnalysisTemplate name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/analysis-templates/{analysis-template} [delete]
func (s *server) deleteAnalysisTemplate(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("analysis-template")

	if err := s.client.Delete(ctx, &v1alpha1.AnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: project,
			Name:      name,
		},
	}); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
