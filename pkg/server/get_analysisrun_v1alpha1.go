package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// nolint: lll
// @id GetAnalysisRun
// @Summary Retrieve an AnalysisRun
// @Description Retrieve an AnalysisRun resource from a project's namespace.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param analysis-run path string true "AnalysisRun name"
// @Produce json
// @Success 200 {object} rolloutsapi.AnalysisRun "AnalysisRun custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.AnalysisRun)"
// @Router /v1beta1/projects/{project}/analysis-runs/{analysis-run} [get]
func (s *server) getAnalysisRun(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("analysis-run")

	run := &rolloutsapi.AnalysisRun{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Namespace: project, Name: name}, run,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, run)
}
