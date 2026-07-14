package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// custom marshaling is not applied when serializing a Warehouse as part of
// a protobuf message, so this helper can be used to compensate for that.
func prepareOutboundWarehouse(w *kargoapi.Warehouse) error {
	type specAlias kargoapi.WarehouseSpec
	specJSON, err := json.Marshal(w.Spec)
	if err != nil {
		return err
	}
	newSpec := specAlias{}
	if err = json.Unmarshal(specJSON, &newSpec); err != nil {
		return err
	}
	w.Spec = kargoapi.WarehouseSpec(newSpec)
	return nil
}

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

	w, err := s.client.Watch(
		ctx,
		&kargoapi.WarehouseList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": name},
	)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch warehouse: %w", err))
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
			if !ConvertAndSendWatchEvent(c, e, (*kargoapi.Warehouse)(nil)) {
				return
			}
		}
	}
}
