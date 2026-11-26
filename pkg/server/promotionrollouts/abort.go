package promotionrollouts

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/user"
)

// @id AbortPromotionRollout
// @Summary Abort a PromotionRollout
// @Description Abort one of a Stage's PromotionRollouts, by its number among
// @Description them. A Pending rollout has created no Promotions, so it ends
// @Description at once, as Aborted. A Running one is asked to stop: it
// @Description creates no more Promotions, those still running are aborted,
// @Description and it ends as Aborted once they have stopped. Until then the
// @Description request is served as the rollout's kargo.akuity.io/abort
// @Description annotation, as a Promotion's is. Asking again changes
// @Description nothing. A rollout that has already finished cannot be
// @Description aborted, and the response is a 409. Stopping a rollout is a
// @Description promotion action: it requires the promote verb on the Stage.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param number path int64 true "PromotionRollout number"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotion-rollouts/{number}/abort [post]
func (h *Handler) abort(c *gin.Context) {
	ctx := c.Request.Context()
	project, stage, raw := c.Param(paramProject), c.Param(paramStage), c.Param(paramNumber)
	number, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || number <= 0 {
		_ = c.Error(libhttp.ErrorStr(fmt.Sprintf("invalid number %q", raw), http.StatusBadRequest))
		return
	}

	// Whoever may promote to the Stage may stop a promotion to it, as with a
	// Promotion. The database carries no authorization of its own, so this
	// is checked before the store is reached.
	if err = h.authorize(ctx, can.Promote().Stage(project, stage)); err != nil {
		_ = c.Error(err)
		return
	}

	var actor string
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
	}
	row, err := h.store.Abort(ctx, project, stage, number, actor)
	if err != nil {
		_ = c.Error(storeError(err, fmt.Sprintf("%s #%d", stage, number)))
		return
	}
	logChange(ctx, "PromotionRollout abort requested", project, row.Name, actor)
	c.Status(http.StatusOK)
}
