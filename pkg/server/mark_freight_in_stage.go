package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/pkg/server/request"
)

// markFreightsCurrentlyInStageRequest is the request body for recording that
// a set of Freights were set as current for stage.
type markFreightsCurrentlyInStageRequest struct {
	// FreightNames is a list of names of freights to be marked as current in stage
	FreightNames []string `json:"freightNames,omitempty"`
} // @name MarkFreightsCurrentlyInStageRequest

// @id MarkFreightsCurrentlyInStage
// @Summary Mark a set of Freights as currently in use by a Stage
// @Description Record that the named Freights are currently in use by the Stage,
// @Description adding the Stage to the Freight's currentlyIn status. Any other
// @Description Freights will have this Stage removed from their currentlyIn.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body markFreightsCurrentlyInStageRequest true "CurrentlyIn"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/stage-freight/{stage}/current [post]
func (s *server) markFreightsCurrentlyInStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stageName := c.Param("stage")

	var req markFreightsCurrentlyInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}

	err := s.freightClient.MarkFreightsCurrentlyInStage(ctx, project, stageName, req.FreightNames)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

// markFreightsVerifiedInStageRequest is the request body for recording that
// Freight has been verified in a Stage.
type markFreightsVerifiedInStageRequest struct {
	// FreightNames is a list of names of freights to be marked as verified in stage
	FreightNames []string `json:"freightNames,omitempty"`
	// VerifiedAt is the time at which the Freight was verified in the Stage. It
	// defaults to the time the request is handled.
	VerifiedAt *metav1.Time `json:"verifiedAt,omitempty"`
} // @name MarkFreightsVerifiedInStageRequest

// @id MarkFreightsVerifiedInStage
// @Summary Mark a set of Freights as verified in a Stage, adding to verifiedIn
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body markFreightsVerifiedInStageRequest true "Verification"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/stage-freight/{stage}/verified [post]
func (s *server) markFreightsVerifiedInStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stageName := c.Param("stage")

	var req markFreightsVerifiedInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}

	verifiedAtTime := time.Now()
	if req.VerifiedAt != nil {
		verifiedAtTime = req.VerifiedAt.Time
	}

	err := s.freightClient.MarkFreightsVerifiedInStage(ctx, project, stageName, req.FreightNames, verifiedAtTime)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

// @id ClearStageFreights
// @Summary Remove all of a Stage's entries from any Freight's status
// @Description Remove the named Stage from all Freights which reference it. This is used by
// @Description the Stage controller to clean up Freight status when a Stage is deleted.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/stage-freight/{stage} [delete]
func (s *server) clearStageFreights(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stageName := c.Param("stage")

	err := s.freightClient.ClearStageFreights(ctx, project, stageName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
