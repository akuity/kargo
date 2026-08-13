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

// @id ListPromotionRequests
// @Summary List PromotionRequests
// @Description List PromotionRequest resources from a project's namespace.
// @Description Returns a PromotionRequestList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Stage filter"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRequestList "PromotionRequestList custom resource"
// @Router /v1beta1/projects/{project}/promotion-requests [get]
func (s *server) listPromotionRequests(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stage := c.Query("stage")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchPromotionRequests(c, project, stage, c.Query("resourceVersion"))
		return
	}

	list := &kargoapi.PromotionRequestList{}
	if err := s.listForWatchSeed(ctx, "promotionrequests", list, client.InNamespace(project)); err != nil {
		_ = c.Error(err)
		return
	}
	if stage != "" {
		list.Items = filterPromotionRequestsByStage(list.Items, stage)
	}

	list.ResourceVersion = normalizeListResourceVersion(list.ResourceVersion)

	// Sort ascending by name. Names embed a ULID, so this is also creation
	// order.
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.PromotionRequest) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}

// filterPromotionRequestsByStage returns PromotionRequests belonging to the
// specified Stage.
//
// Filtering happens in-process for the same reason it does for Promotions: the
// watch-seed list goes through listForWatchSeed's uncached reader, which cannot
// serve controller-runtime field indexes.
func filterPromotionRequestsByStage(
	promotionRequests []kargoapi.PromotionRequest,
	stage string,
) []kargoapi.PromotionRequest {
	filtered := make([]kargoapi.PromotionRequest, 0, len(promotionRequests))
	for _, promotionRequest := range promotionRequests {
		if promotionRequest.Spec.Stage == stage {
			filtered = append(filtered, promotionRequest)
		}
	}
	return filtered
}

// watchPromotionRequests streams PromotionRequest changes through the REST SSE
// endpoint.
func (s *server) watchPromotionRequests(c *gin.Context, project, stage, resourceVersion string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// As with Promotions, the watch API cannot filter by stage with a field
	// selector, so events are filtered here.
	w, err := s.client.Watch(
		ctx,
		&kargoapi.PromotionRequestList{},
		buildWatchListOptions(project, resourceVersion)...,
	)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch promotion requests: %w", err))
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

			promotionRequest, ok := ConvertWatchEventObject(c, e, (*kargoapi.PromotionRequest)(nil))
			if !ok {
				continue
			}

			eventType := e.Type
			if stage != "" {
				var send bool
				eventType, send = FilteredWatchEventType(e.Type, promotionRequest.Spec.Stage == stage)
				if !send {
					continue
				}
			}

			if !SendSSEWatchEvent(c, eventType, promotionRequest) {
				return
			}
		}
	}
}
