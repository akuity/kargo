package targetstatuses

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/database/targetstatusstore"
)

// @id GetTargetStatusSummary
// @Summary Summarize Target statuses
// @Description Count a project's Targets by where they stand with the
// @Description Stages that govern them: for each Stage, how many of the
// @Description Targets it governs are in each state; and for the project,
// @Description how many Targets are in each state, a Target governed by
// @Description several Stages counted once, under the worst state any of
// @Description them reports. The states and the entries counted are those
// @Description the Target status list serves, and are counted in the
// @Description database rather than read. Targets behind their Stage's
// @Description current Freight are counted as well, whatever their state.
// @Description Target health is not recorded, so nothing here reports it.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only count the Targets this Stage governs"
// @Produce json
// @Success 200 {object} Summary "Counts of Targets by state"
// @Router /v1beta1/projects/{project}/target-statuses/summary [get]
func (h *Handler) summarize(c *gin.Context) {
	ctx := c.Request.Context()
	fleet, err := h.govern(ctx, c.Param(paramProject), c.Query(queryStage))
	if err != nil {
		_ = c.Error(err)
		return
	}
	stored, err := h.store.Summarize(ctx, fleet.scope(fleet.pairs))
	if err != nil {
		_ = c.Error(err)
		return
	}
	summary := Summary{
		Targets: int64(len(fleet.targets)),
		States:  stored.Targets.States,
		Behind:  stored.Targets.Behind,
		Stages:  make([]StageSummary, len(fleet.stages)),
	}
	for i, stage := range fleet.stages {
		counts, ok := stored.Stages[stage.id]
		if !ok {
			// A Stage that governs no Targets is still one of the Project's:
			// it is listed with every State at zero.
			counts = targetstatusstore.Counts{States: map[targetstatusstore.State]int64{}}
			for _, state := range targetstatusstore.States() {
				counts.States[state] = 0
			}
		}
		summary.Stages[i] = StageSummary{
			Stage:  stage.name,
			States: counts.States,
			Behind: counts.Behind,
		}
	}
	c.JSON(http.StatusOK, summary)
}
