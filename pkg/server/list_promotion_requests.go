package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
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

	if s.store == nil {
		_ = c.Error(errDatabaseNotConfigured)
		return
	}

	watchMode := c.Query("watch") == trueStr
	verb := "list"
	if watchMode {
		verb = "watch"
	}
	if err := s.authorizeStoreRead(ctx, verb, "promotionrequests", project, ""); err != nil {
		_ = c.Error(err)
		return
	}

	snapshot := func(ctx context.Context) ([]kargoapi.PromotionRequest, error) {
		return s.listPromotionRequestsFromStore(ctx, project, stage)
	}

	if watchMode {
		servePolledWatch(c, s.storePollInterval, c.Query("resourceVersion"), polledWatch[kargoapi.PromotionRequest]{
			snapshot: snapshot,
			name:     func(request kargoapi.PromotionRequest) string { return request.Name },
			version:  func(request kargoapi.PromotionRequest) string { return request.ResourceVersion },
		})
		return
	}

	items, err := snapshot(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}
	list := &kargoapi.PromotionRequestList{Items: items}
	versions := make([]string, len(items))
	for i, item := range items {
		versions[i] = item.ResourceVersion
	}
	list.ResourceVersion = maxResourceVersion(versions...)

	c.JSON(http.StatusOK, list)
}

// listPromotionRequestsFromStore reads a project's PromotionRequests, or only
// those of one Stage, in creation order. The result is never nil.
func (s *server) listPromotionRequestsFromStore(
	ctx context.Context,
	project string,
	stage string,
) ([]kargoapi.PromotionRequest, error) {
	var (
		snapshots []database.PromotionRequestSnapshot
		err       error
	)
	if stage == "" {
		snapshots, err = s.store.ListPromotionRequests(ctx, project)
	} else {
		snapshots, err = s.store.ListPromotionRequestsByStage(ctx, database.ListPromotionRequestsByStageParams{
			ProjectName: project,
			Stage:       stage,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("error listing PromotionRequests in Project %q: %w", project, err)
	}
	return database.PromotionRequestsFromSnapshots(snapshots), nil
}
