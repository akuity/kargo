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
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) ListStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	warehouses := req.Msg.GetFreightOrigins()

	// The ConnectRPC response carries resource_version for list-then-watch,
	// so the list call is done inline here rather than via
	// api.ListStagesByWarehouses (which does not surface the list's
	// ResourceVersion). Filtering is delegated to api.StageMatchesAnyWarehouse
	// so the matching logic stays in one place.
	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	items := list.Items
	if len(warehouses) > 0 {
		items = items[:0:0]
		for _, stage := range list.Items {
			if api.StageMatchesAnyWarehouse(&stage, warehouses) {
				items = append(items, stage)
			}
		}
	}

	slices.SortFunc(items, func(a, b kargoapi.Stage) int {
		return strings.Compare(a.Name, b.Name)
	})

	summary := req.Msg.GetSummary()
	stages := make([]*kargoapi.Stage, len(items))
	for idx := range items {
		if summary {
			stripStageForSummary(&items[idx])
		}
		stages[idx] = &items[idx]
	}
	return connect.NewResponse(&svcv1alpha1.ListStagesResponse{
		Stages:          stages,
		ResourceVersion: list.ResourceVersion,
	}), nil
}

// stripStageForSummary mutates the Stage in place, clearing the heavy
// payload fields that list and graph views do not need. The surviving
// shape still preserves has-verification and promotion-step-count
// information (via stage.Spec.Verification != nil and
// len(stage.Spec.PromotionTemplate.Spec.Steps)), so callers do not have
// to refetch via GetStage for those bits.
//
// Stripped fields:
//   - status.freightHistory truncated to the current element (index 0)
//   - spec.promotionTemplate.spec.steps[*].config cleared (kind/as/name kept)
//   - status.health.output cleared (use GetStageHealthOutputs for lazy fetch)
func stripStageForSummary(stage *kargoapi.Stage) {
	if stage == nil {
		return
	}
	if len(stage.Status.FreightHistory) > 1 {
		stage.Status.FreightHistory = stage.Status.FreightHistory[:1]
	}
	if stage.Spec.PromotionTemplate != nil {
		for i := range stage.Spec.PromotionTemplate.Spec.Steps {
			stage.Spec.PromotionTemplate.Spec.Steps[i].Config = nil
		}
	}
	if stage.Status.Health != nil {
		stage.Status.Health.Output = nil
	}
}

// @id ListStages
// @Summary List Stages
// @Description List Stage resources from a project's namespace. Returns a
// @Description StageList resource. Pass summary=true to receive a lightweight
// @Description projection for list and graph views.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freightOrigins query []string false "Warehouse names to filter Stages by"
// @Param summary query bool false "Strip heavy fields from each Stage"
// @Success 200 {object} kargoapi.StageList "StageList custom resource (github.com/akuity/kargo/api/v1alpha1.StageList)"
// @Router /v1beta1/projects/{project}/stages [get]
func (s *server) listStages(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	warehouses := c.QueryArray("freightOrigins")
	summary := c.Query("summary") == trueStr

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchStages(c, project, warehouses, summary)
		return
	}

	items, err := api.ListStagesByWarehouses(
		ctx,
		s.client,
		project,
		&api.ListStagesOptions{Warehouses: warehouses},
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if summary {
		for i := range items {
			stripStageForSummary(&items[i])
		}
	}

	c.JSON(http.StatusOK, &kargoapi.StageList{Items: items})
}

func (s *server) watchStages(
	c *gin.Context,
	project string,
	warehouses []string,
	summary bool,
) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, client.InNamespace(project))
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch stages: %w", err))
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

			stage, ok := convertWatchEventObject(c, e, (*kargoapi.Stage)(nil))
			if !ok {
				continue
			}

			if len(warehouses) > 0 && !api.StageMatchesAnyWarehouse(stage, warehouses) {
				continue
			}

			if summary {
				stripStageForSummary(stage)
			}

			if !sendSSEWatchEvent(c, e.Type, stage) {
				return
			}
		}
	}
}
