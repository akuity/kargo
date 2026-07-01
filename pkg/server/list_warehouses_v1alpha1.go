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
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) ListWarehouses(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListWarehousesRequest],
) (*connect.Response[svcv1alpha1.ListWarehousesResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list kargoapi.WarehouseList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}

	slices.SortFunc(list.Items, func(a, b kargoapi.Warehouse) int {
		return strings.Compare(a.Name, b.Name)
	})

	warehouses := make([]*kargoapi.Warehouse, len(list.Items))
	for idx := range list.Items {
		warehouses[idx] = &list.Items[idx]
		// Necessary because serializing a Warehouse as part of a protobuf message
		// does not apply custom marshaling. The call to this helper compensates for
		// that.
		if err := prepareOutboundWarehouse(warehouses[idx]); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&svcv1alpha1.ListWarehousesResponse{
		Warehouses: warehouses,
	}), nil
}

// @id ListWarehouses
// @Summary List Warehouses
// @Description List Warehouse resources from a project's namespace. Returns a
// @Description WarehouseList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} kargoapi.WarehouseList "WarehouseList custom resource"
// @Router /v1beta1/projects/{project}/warehouses [get]
func (s *server) listWarehouses(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchWarehouses(c, project, c.Query("resourceVersion"))
		return
	}

	list := &kargoapi.WarehouseList{}
	if err := s.listForWatchSeed(ctx, "warehouses", list, client.InNamespace(project)); err != nil {
		_ = c.Error(err)
		return
	}
	list.ResourceVersion = normalizeListResourceVersion(list.ResourceVersion)

	c.JSON(http.StatusOK, list)
}

// watchWarehouses streams Warehouse changes through the REST SSE endpoint.
func (s *server) watchWarehouses(c *gin.Context, project string, resourceVersion string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	w, err := s.client.Watch(
		ctx,
		&kargoapi.WarehouseList{},
		buildWatchListOptions(project, resourceVersion)...,
	)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch warehouses: %w", err))
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
			// convertAndSendWatchEvent surfaces watch.Error events itself, so
			// no separate errorFromWatchEvent check is needed here (unlike the
			// filtered Stage/Promotion handlers, which inspect the event before
			// applying their event-type filter).
			if !ConvertAndSendWatchEvent(c, e, (*kargoapi.Warehouse)(nil)) {
				return
			}
		}
	}
}
