package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

// @id ListStages
// @Summary List Stages
// @Description List Stage resources from a project's namespace. Returns a
// @Description StageList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freightOrigins query []string false "Warehouse names to filter Stages by"
// @Success 200 {object} kargoapi.StageList "StageList custom resource (github.com/akuity/kargo/api/v1alpha1.StageList)"
// @Router /v1beta1/projects/{project}/stages [get]
func (s *server) listStages(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	warehouses := c.QueryArray("freightOrigins")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchStages(c, project, warehouses, c.Query("resourceVersion"))
		return
	}

	list, err := s.listStagesByWarehouses(ctx, project, warehouses)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, list)
}

// watchStages streams Stage changes through the REST SSE endpoint.
func (s *server) watchStages(c *gin.Context, project string, warehouses []string, resourceVersion string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	w, err := s.client.Watch(
		ctx,
		&kargoapi.StageList{},
		buildWatchListOptions(project, resourceVersion)...,
	)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch stages: %w", err))
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

			stage, ok := ConvertWatchEventObject(c, e, (*kargoapi.Stage)(nil))
			if !ok {
				continue
			}

			eventType := e.Type
			if len(warehouses) > 0 {
				var send bool
				eventType, send = FilteredWatchEventType(
					e.Type,
					api.StageMatchesAnyWarehouse(stage, warehouses),
				)
				if !send {
					continue
				}
			}

			if !SendSSEWatchEvent(c, eventType, stage) {
				return
			}
		}
	}
}

// listStagesByWarehouses lists Stages in the given project, optionally
// filtered to those that request Freight from at least one of the specified
// warehouses (directly or through upstream stages). When warehouses is empty,
// all Stages are returned. The returned StageList carries an effective
// ResourceVersion derived from the list response or listed items.
func (s *server) listStagesByWarehouses(
	ctx context.Context,
	project string,
	warehouses []string,
) (*kargoapi.StageList, error) {
	var list kargoapi.StageList
	if err := s.listForWatchSeed(ctx, "stages", &list, client.InNamespace(project)); err != nil {
		return nil, err
	}
	list.ResourceVersion = normalizeListResourceVersion(list.ResourceVersion)
	if len(warehouses) == 0 {
		return &list, nil
	}
	stages := make([]kargoapi.Stage, 0, len(list.Items))
	for _, stage := range list.Items {
		if api.StageMatchesAnyWarehouse(&stage, warehouses) {
			stages = append(stages, stage)
		}
	}
	list.Items = stages
	return &list, nil
}
