package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// nolint: lll
// @id GetAnalysisTemplate
// @Summary Retrieve an AnalysisTemplate
// @Description Retrieve an AnalysisTemplate resource from a project's
// @Description namespace.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param analysis-template path string true "AnalysisTemplate name"
// @Produce json
// @Success 200 {object} rolloutsapi.AnalysisTemplate "AnalysisTemplate custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.AnalysisTemplate)"
// @Router /v1beta1/projects/{project}/analysis-templates/{analysis-template} [get]
func (s *server) getAnalysisTemplate(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("analysis-template")

	template := &rolloutsapi.AnalysisTemplate{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Namespace: project, Name: name}, template,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, template)
}
