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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	list, err := api.ListStagesByWarehouses(
		ctx,
		s.client,
		project,
		&api.ListStagesOptions{Warehouses: warehouses},
	)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	items := list.Items

	slices.SortFunc(items, func(a, b kargoapi.Stage) int {
		return strings.Compare(a.Name, b.Name)
	})

	summary := req.Msg.GetSummary()
	stages := make([]*kargoapi.Stage, len(items))
	for idx := range items {
		if summary {
			api.StripStageForSummary(&items[idx])
		}
		stages[idx] = &items[idx]
	}
	return connect.NewResponse(&svcv1alpha1.ListStagesResponse{
		Stages:          stages,
		ResourceVersion: list.ResourceVersion,
	}), nil
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
// @Param watch query bool false "Stream Stage changes as Server-Sent Events instead of returning a list"
// @Param resourceVersion query string false "When watch=true, resume after this ResourceVersion"
// @Success 200 {object} kargoapi.StageList "StageList custom resource (github.com/akuity/kargo/api/v1alpha1.StageList)"
// @Router /v1beta1/projects/{project}/stages [get]
func (s *server) listStages(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	warehouses := c.QueryArray("freightOrigins")
	summary := c.Query("summary") == trueStr

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchStages(c, project, warehouses, summary, c.Query("resourceVersion"))
		return
	}

	list, err := api.ListStagesByWarehouses(
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
		for i := range list.Items {
			api.StripStageForSummary(&list.Items[i])
		}
	}

	c.JSON(http.StatusOK, list)
}

func (s *server) watchStages(
	c *gin.Context,
	project string,
	warehouses []string,
	summary bool,
	resourceVersion string,
) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	listOpts := []client.ListOption{client.InNamespace(project)}
	if resourceVersion != "" {
		listOpts = append(
			listOpts,
			&client.ListOptions{Raw: &metav1.ListOptions{ResourceVersion: resourceVersion}},
		)
	}
	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, listOpts...)
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
				// StripStageForSummary mutates in place; copy first so
				// shaping the response does not depend on the watch
				// object's ownership semantics.
				stage = stage.DeepCopy()
				api.StripStageForSummary(stage)
			}

			if !sendSSEWatchEvent(c, e.Type, stage) {
				return
			}
		}
	}
}
