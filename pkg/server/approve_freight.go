package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

// @id ApproveFreight
// @Summary Approve Freight for promotion to a Stage
// @Description Approve Freight for promotion to a Stage.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param stage query string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/approve [post]
func (s *server) approveFreight(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightNameOrAlias := c.Param("freight-name-or-alias")
	stageName := c.Query("stage")

	if stageName == "" {
		_ = c.Error(libhttp.Error(
			errors.New("stage query parameter is required"),
			http.StatusBadRequest,
		))
		return
	}

	if err := s.authorize(ctx, can.Promote().Stage(project, stageName)); err != nil {
		_ = c.Error(err)
		return
	}

	if err := s.storageClient.ApproveFreight(ctx, project, stageName, freightNameOrAlias); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
