package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) GetWarehouse(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetWarehouseRequest],
) (*connect.Response[svcv1alpha1.GetWarehouseResponse], error) {
	project := req.Msg.GetProject()

	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}
	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	// Get the Warehouse from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "Warehouse",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: project}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// nolint:staticcheck
			err = fmt.Errorf("Warehouse %q not found in project %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	w, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.Warehouse{},
	)
	if err != nil {
		return nil, err
	}
	if w != nil {
		// Necessary because serializing a Warehouse as part of a protobuf message
		// does not apply custom marshaling. The call to this helper compensates for
		// that.
		if err := prepareOutboundWarehouse(w); err != nil {
			return nil, err
		}
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetWarehouseResponse{
			Result: &svcv1alpha1.GetWarehouseResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetWarehouseResponse{
		Result: &svcv1alpha1.GetWarehouseResponse_Warehouse{Warehouse: w},
	}), nil
}

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
// @Success 200 {object} object "Warehouse custom resource (github.com/akuity/kargo/api/v1alpha1.Warehouse)"
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
			if !convertAndSendWatchEvent(c, e, (*kargoapi.Warehouse)(nil)) {
				return
			}
		}
	}
}
