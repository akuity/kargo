package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) ListPromotions(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListPromotionsRequest],
) (*connect.Response[svcv1alpha1.ListPromotionsResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	stage := req.Msg.GetStage()

	var list kargoapi.PromotionList
	opts := []client.ListOption{
		client.InNamespace(project),
	}
	if stage != "" {
		opts = append(opts, client.MatchingFields{indexer.PromotionsByStageField: stage})
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list promotions: %w", err)
	}

	slices.SortFunc(list.Items, api.ComparePromotionByPhaseAndCreationTime)

	promotions := make([]*kargoapi.Promotion, len(list.Items))
	for idx := range list.Items {
		promotions[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListPromotionsResponse{
		Promotions: promotions,
	}), nil
}

// @id ListPromotions
// @Summary List Promotions
// @Description List Promotion resources from a project's namespace. Returns a
// @Description PromotionList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Stage filter"
// @Produce json
// @Success 200 {object} object "PromotionList custom resource (github.com/akuity/kargo/api/v1alpha1.PromotionList)"
// @Router /v1beta1/projects/{project}/promotions [get]
func (s *server) listPromotions(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stage := c.Query("stage")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchPromotions(c, project, stage)
		return
	}

	list := &kargoapi.PromotionList{}
	opts := []client.ListOption{
		client.InNamespace(project),
	}
	if stage != "" {
		opts = append(opts, client.MatchingFields{indexer.PromotionsByStageField: stage})
	}
	if err := s.client.List(ctx, list, opts...); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.Promotion) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}

func (s *server) watchPromotions(c *gin.Context, project, stage string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Note: We can't filter by stage using field selector in the watch API.
	// The indexer is for List operations only. We filter events client-side.
	w, err := s.client.Watch(ctx, &kargoapi.PromotionList{}, client.InNamespace(project))
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch promotions: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	setSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !writeSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}

			promotion, ok := convertWatchEventObject(c, e, (*kargoapi.Promotion)(nil))
			if !ok {
				continue
			}

			// Filter by stage if specified (client-side filtering)
			if stage != "" && promotion.Spec.Stage != stage {
				continue
			}

			if !sendSSEWatchEvent(c, e.Type, promotion) {
				return
			}
		}
	}
}
