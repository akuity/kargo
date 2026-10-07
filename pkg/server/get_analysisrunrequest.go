package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// nolint: lll
// @id GetAnalysisRunRequest
// @Summary Retrieve an AnalysisRunRequest
// @Description Retrieve an AnalysisRunRequest resource from a project's namespace.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param analysis-run-request path string true "AnalysisRunRequest name"
// @Produce json
// @Success 200 {object} kargoapi.AnalysisRunRequest "AnalysisRunRequest custom resource"
// @Router /v1beta1/projects/{project}/analysis-run-requests/{analysis-run-request} [get]
func (s *server) getAnalysisRunRequest(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("analysis-run-request")

	req := &kargoapi.AnalysisRunRequest{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Namespace: project, Name: name}, req,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, req)
}
