package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

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

	watcher, err := s.watchK8sResources[*kargoapi.Warehouse](
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
	s.watchResources(c, watcher, nil)
}
