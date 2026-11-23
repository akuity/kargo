package targetstatuses

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
)

// Promotion is a Promotion to a Target, as the database recorded it. The
// record outlives the Promotion in Kubernetes.
type Promotion struct {
	// Name is the Promotion's name.
	Name string `json:"name"`
	// Stage is the name of the Stage that promoted.
	Stage string `json:"stage"`
	// Freight is the name of the Freight promoted.
	Freight string `json:"freight"`
	// Phase is the Promotion's phase. One that has yet to be picked up is
	// Pending.
	Phase kargoapi.PromotionPhase `json:"phase"`
	// Rollout is the number, among the Stage's, of the PromotionRollout that
	// created the Promotion. It is absent for one created on its own.
	Rollout int64 `json:"rollout,omitempty"`
	// CreatedAt is when the Promotion was created.
	CreatedAt time.Time `json:"createdAt"`
	// FinishedAt is when the Promotion finished. It is absent until it has.
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
} // @name TargetPromotion

// PromotionList is one page of the Promotions to a Target.
type PromotionList struct {
	// Items are the Promotions, newest first.
	Items []Promotion `json:"items"`
	// Continue is set when there may be more: pass it as continue to read
	// the next page.
	Continue string `json:"continue,omitempty"`
} // @name TargetPromotionList

// @id ListTargetPromotions
// @Summary List a Target's Promotions
// @Description List the Promotions to one of a project's Targets, newest
// @Description first, one page at a time: those of every Stage, or of one.
// @Description They are read from the database's record of Promotions,
// @Description which outlives the Promotions themselves, so the list
// @Description reaches further back than the project's Promotions do. A
// @Description Target is recorded by name, so this is the history of the
// @Description name: a Target that has had no Promotion, or that does not
// @Description exist, has an empty one.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param target-name path string true "Target name"
// @Param stage query string false "Only Promotions of this Stage"
// @Param continue query string false "The previous page's continue, to read the page after it; resend stage"
// @Param limit query int false "The page size, at most 500; 100 if omitted"
// @Produce json
// @Success 200 {object} PromotionList "A page of the Target's Promotions"
// @Router /v1beta1/projects/{project}/targets/{target-name}/promotions [get]
func (h *Handler) listPromotions(c *gin.Context) {
	limit, err := parseLimit(c.Query(queryLimit))
	if err != nil {
		_ = c.Error(err)
		return
	}
	opts := targetstatusstore.PromotionListOptions{
		Stage: c.Query(queryStage),
		Limit: limit,
	}
	if token := c.Query(queryContinue); token != "" {
		before := promotionCursor{}
		if err = decodeToken(token, &before); err != nil {
			_ = c.Error(err)
			return
		}
		opts.Before = &targetstatusstore.PromotionCursor{
			CreatedAt: before.CreatedAt,
			ID:        before.ID,
		}
	}
	promotions, err := h.store.ListPromotions(
		c.Request.Context(),
		c.Param(paramProject),
		c.Param(paramTarget),
		opts,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	list := PromotionList{Items: make([]Promotion, len(promotions))}
	for i, promotion := range promotions {
		list.Items[i] = presentPromotion(promotion)
	}
	// A full page may have more after it; a short page is the last.
	if len(promotions) > 0 && len(promotions) == int(limit) {
		last := promotions[len(promotions)-1]
		list.Continue = encodeToken(promotionCursor{CreatedAt: last.CreatedAt.UTC(), ID: last.ID})
	}
	c.JSON(http.StatusOK, list)
}

// promotionCursor marks a place in a Target's Promotions by the Promotion
// at it: the last of the previous page. It is what a continue token holds.
type promotionCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

// presentPromotion returns a stored Promotion as the API serves it. Times
// are served in UTC, whatever zone the database's driver reads them in.
func presentPromotion(stored targetstatusstore.Promotion) Promotion {
	promotion := Promotion{
		Name:      stored.Name,
		Stage:     stored.Stage,
		Freight:   stored.Freight,
		Phase:     kargoapi.PromotionPhase(stored.Phase),
		Rollout:   stored.Rollout,
		CreatedAt: stored.CreatedAt.UTC(),
	}
	if stored.FinishedAt != nil {
		finished := stored.FinishedAt.UTC()
		promotion.FinishedAt = &finished
	}
	return promotion
}
