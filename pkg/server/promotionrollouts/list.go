package promotionrollouts

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// maxPageSize is the most PromotionRollouts one list returns, whatever limit
// is asked for.
const maxPageSize = 500

// The query parameters of the list endpoint.
const (
	queryStage    = "stage"
	queryFreight  = "freight"
	queryLatest   = "latest"
	queryAfter    = "after"
	queryContinue = "continue"
	queryLimit    = "limit"
)

// @id ListPromotionRollouts
// @Summary List PromotionRollouts
// @Description List a project's PromotionRollouts, one page at a time, as
// @Description a PromotionRolloutList resource. A Project or a Stage may
// @Description have had any number of rollouts, so a list is never more
// @Description than one page, newest first: the Project's, or a Stage's,
// @Description or a Stage's of one piece of Freight. When a page is full,
// @Description the list's metadata.continue names where the next page
// @Description starts; pass it as continue to read on. With after, a
// @Description Stage's rollouts numbered above it are listed oldest first
// @Description instead. With latest, the list is instead each Stage's
// @Description latest rollouts -- the one it is promoting through, if any,
// @Description and the newest that has finished, if any -- which is what a
// @Description reader of a Stage's state wants without reading its history.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only PromotionRollouts of this Stage"
// @Param freight query string false "Only PromotionRollouts of this Freight; requires stage"
// @Param latest query boolean false "Each Stage's latest rollouts, or the named Stage's; takes only stage besides"
// @Param after query int64 false "Only the Stage's PromotionRollouts numbered above this, oldest first; requires stage"
// @Param continue query string false "The metadata.continue of the previous page, to read the page after it"
// @Param limit query int false "The page size, at most 500; 50 if omitted"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRolloutList "PromotionRolloutList resource"
// @Router /v1beta1/projects/{project}/promotion-rollouts [get]
func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)
	stage := c.Query(queryStage)

	var (
		stored []promotionrolloutstore.Rollout
		err    error
	)
	latest, _ := strconv.ParseBool(c.Query(queryLatest))
	if latest {
		if c.Query(queryFreight) != "" || c.Query(queryAfter) != "" || c.Query(queryContinue) != "" {
			_ = c.Error(libhttp.ErrorStr(
				"latest cannot be combined with freight, after or continue",
				http.StatusBadRequest,
			))
			return
		}
		stored, err = h.store.ListLatest(ctx, project, stage)
	} else {
		var opts promotionrolloutstore.ListOptions
		if opts, err = h.listOptions(c); err != nil {
			_ = c.Error(err)
			return
		}
		stored, err = h.store.List(ctx, project, opts)
	}
	if err != nil {
		_ = c.Error(err)
		return
	}

	rollouts, err := h.present(ctx, project, stored)
	if err != nil {
		_ = c.Error(err)
		return
	}
	list := &kargoapi.PromotionRolloutList{Items: make([]kargoapi.PromotionRollout, len(rollouts))}
	for i, rollout := range rollouts {
		list.Items[i] = *rollout
	}
	// A full page may have more after it. The page is continued from its
	// last rollout, which the token names; a short page is the last.
	if !latest && len(list.Items) > 0 && len(list.Items) == c.GetInt(queryLimit) {
		list.ListMeta = metav1.ListMeta{Continue: list.Items[len(list.Items)-1].Name}
	}
	c.JSON(http.StatusOK, list)
}

// listOptions reads a page's options from the request: which rollouts, and
// from where. A continue token names the rollout the previous page ended
// at, which is read back to find where this one starts. The limit the page
// was read with is kept on the context for the handler to tell a full page
// by.
func (h *Handler) listOptions(c *gin.Context) (promotionrolloutstore.ListOptions, error) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)
	opts := promotionrolloutstore.ListOptions{
		Stage:   c.Query(queryStage),
		Freight: c.Query(queryFreight),
		Limit:   promotionrolloutstore.DefaultPageSize,
	}
	if opts.Freight != "" && opts.Stage == "" {
		return opts, libhttp.ErrorStr("freight requires stage", http.StatusBadRequest)
	}
	if raw := c.Query(queryLimit); raw != "" {
		limit, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || limit <= 0 {
			return opts, libhttp.ErrorStr(fmt.Sprintf("invalid limit %q", raw), http.StatusBadRequest)
		}
		opts.Limit = int32(min(limit, maxPageSize))
	}
	c.Set(queryLimit, int(opts.Limit))

	var cursor *promotionrolloutstore.Cursor
	if token := c.Query(queryContinue); token != "" {
		rollout, err := h.store.Get(ctx, project, token)
		if errors.Is(err, database.ErrNotFound) {
			return opts, libhttp.ErrorStr(fmt.Sprintf("unknown continue token %q", token), http.StatusBadRequest)
		}
		if err != nil {
			return opts, err
		}
		at := promotionrolloutstore.CursorOf(rollout)
		cursor = &at
	}
	if raw := c.Query(queryAfter); raw != "" {
		if opts.Stage == "" || opts.Freight != "" {
			return opts, libhttp.ErrorStr("after requires stage and excludes freight", http.StatusBadRequest)
		}
		after, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || after < 0 {
			return opts, libhttp.ErrorStr(fmt.Sprintf("invalid after %q", raw), http.StatusBadRequest)
		}
		// A page continues after its last rollout, which is past the number
		// asked for.
		if cursor == nil {
			cursor = &promotionrolloutstore.Cursor{Number: after}
		}
		opts.After = cursor
		return opts, nil
	}
	opts.Before = cursor
	return opts, nil
}
