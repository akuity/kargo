package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// @id ListPromotions
// @Summary List Promotions
// @Description List Promotion resources from a project's namespace. Returns a
// @Description PromotionList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Stage filter"
// @Produce json
// @Success 200 {object} kargoapi.PromotionList "PromotionList custom resource"
// @Router /v1beta1/projects/{project}/promotions [get]
func (s *server) listPromotions(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stage := c.Query("stage")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchPromotions(c, project, stage, c.Query("resourceVersion"))
		return
	}

	list := &kargoapi.PromotionList{}
	if err := s.listForWatchSeed(ctx, "promotions", list, client.InNamespace(project)); err != nil {
		_ = c.Error(err)
		return
	}
	if stage != "" {
		list.Items = filterPromotionsByStage(list.Items, stage)
	}

	list.ResourceVersion = normalizeListResourceVersion(list.ResourceVersion)

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.Promotion) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}

// filterPromotionsByStage returns Promotions that target the specified Stage.
//
// Stage filtering is done in-process rather than via the PromotionsByStage
// field index because the watch-seed list goes through listForWatchSeed's
// uncached reader, which cannot serve controller-runtime field indexes. This
// over-fetches the whole namespace, but it matches the (also unfiltered)
// follow-up watch and keeps the returned ResourceVersion watchable.
func filterPromotionsByStage(promotions []kargoapi.Promotion, stage string) []kargoapi.Promotion {
	filtered := make([]kargoapi.Promotion, 0, len(promotions))
	for _, promotion := range promotions {
		if promotion.Spec.Stage == stage {
			filtered = append(filtered, promotion)
		}
	}
	return filtered
}

// watchPromotions streams Promotion changes through the REST SSE endpoint.
func (s *server) watchPromotions(c *gin.Context, project, stage, resourceVersion string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Note: We can't filter by stage using field selector in the watch API.
	// The indexer is for List operations only. We filter events client-side.
	w, err := s.client.Watch(
		ctx,
		&kargoapi.PromotionList{},
		buildWatchListOptions(project, resourceVersion)...,
	)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch promotions: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	SetSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !WriteSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}
			if watchErr := ErrorFromWatchEvent(e); watchErr != nil {
				SendSSEWatchError(c, watchErr)
				return
			}

			promotion, ok := ConvertWatchEventObject(c, e, (*kargoapi.Promotion)(nil))
			if !ok {
				continue
			}

			eventType := e.Type
			if stage != "" {
				var send bool
				eventType, send = FilteredWatchEventType(e.Type, promotion.Spec.Stage == stage)
				if !send {
					continue
				}
			}

			if !SendSSEWatchEvent(c, eventType, promotion) {
				return
			}
		}
	}
}
