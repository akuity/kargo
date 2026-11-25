package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// @id GetWarehouse
// @Summary Retrieve a Warehouse
// @Description Retrieve a Warehouse resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param warehouse path string true "Warehouse name"
// @Produce json
// @Success 200 {object} kargoapi.Warehouse "Warehouse custom resource (github.com/akuity/kargo/api/v1alpha1.Warehouse)"
// @Router /v1beta1/projects/{project}/warehouses/{warehouse} [get]
func (s *server) getWarehouse(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("warehouse")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchWarehouse(c, project, name)
		return
	}

	warehouse := &kargoapi.Warehouse{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		warehouse,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, warehouse)
}

func (s *server) watchWarehouse(c *gin.Context, project, name string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Validate that the warehouse exists
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: project,
		Name:      name,
	}, &kargoapi.Warehouse{}); err != nil {
		_ = c.Error(err)
		return
	}

	watcher, err := s.watchK8sResources[*kargoapi.Warehouse](
		ctx,
		&kargoapi.WarehouseList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": name},
	)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch stage: %w", err))
		return
	}
	s.watchResources(c, watcher, nil)
}
