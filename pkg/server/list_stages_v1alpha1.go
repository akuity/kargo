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

	list, err := s.listStagesByWarehouses(ctx, project, warehouses)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	slices.SortFunc(list.Items, func(a, b kargoapi.Stage) int {
		return strings.Compare(a.Name, b.Name)
	})

	stages := make([]*kargoapi.Stage, len(list.Items))
	rvs := make([]string, len(list.Items))
	for idx := range list.Items {
		stages[idx] = &list.Items[idx]
		rvs[idx] = list.Items[idx].ResourceVersion
	}
	return connect.NewResponse(&svcv1alpha1.ListStagesResponse{
		Stages:          stages,
		ResourceVersion: effectiveResourceVersion(list.ResourceVersion, rvs),
	}), nil
}

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

func (s *server) watchStages(c *gin.Context, project string, warehouses []string, resourceVersion string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	watchOpts := []client.ListOption{client.InNamespace(project)}
	if resourceVersion != "" {
		watchOpts = append(watchOpts, &client.ListOptions{
			Raw: &metav1.ListOptions{ResourceVersion: resourceVersion},
		})
	}
	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, watchOpts...)
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

			if len(warehouses) > 0 && !stageMatchesAnyWarehouse(stage, warehouses) {
				continue
			}

			if !sendSSEWatchEvent(c, e.Type, stage) {
				return
			}
		}
	}
}

// listStagesByWarehouses lists Stages in the given project, optionally
// filtered to those that request Freight from at least one of the specified
// warehouses (directly or through upstream stages). When warehouses is empty,
// all Stages are returned. The returned StageList carries the ResourceVersion
// from the underlying Kubernetes List call.
func (s *server) listStagesByWarehouses(
	ctx context.Context,
	project string,
	warehouses []string,
) (*kargoapi.StageList, error) {
	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, err
	}
	if len(warehouses) == 0 {
		return &list, nil
	}
	var stages []kargoapi.Stage
	for _, stage := range list.Items {
		if stageMatchesAnyWarehouse(&stage, warehouses) {
			stages = append(stages, stage)
		}
	}
	list.Items = stages
	return &list, nil
}

// stageMatchesAnyWarehouse returns true if the Stage requests Freight that
// originated from at least one of the specified warehouses, either directly
// or through upstream stages.
func stageMatchesAnyWarehouse(stage *kargoapi.Stage, warehouses []string) bool {
	for _, req := range stage.Spec.RequestedFreight {
		if req.Origin.Kind == kargoapi.FreightOriginKindWarehouse &&
			slices.Contains(warehouses, req.Origin.Name) {
			return true
		}
	}
	return false
}
