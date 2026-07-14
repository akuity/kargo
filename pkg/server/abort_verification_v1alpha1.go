package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/api"
)

// @id AbortVerification
// @Summary Abort a running Verification process
// @Description Abort a running Verification process.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/verification/abort [post]
func (s *server) abortVerification(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stage := c.Param("stage")

	if err := api.AbortStageFreightVerification(
		ctx,
		s.client,
		client.ObjectKey{Namespace: project, Name: stage},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
