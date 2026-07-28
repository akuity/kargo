package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/api"
)

// @id Reverify
// @Summary Reverify Freight
// @Description Trigger re-verification of the Freight currently in use by a
// @Description Stage.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/verification [post]
func (s *server) reverify(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stage := c.Param("stage")

	if err := api.ReverifyStageFreight(
		ctx,
		s.client,
		client.ObjectKey{Namespace: project, Name: stage},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
