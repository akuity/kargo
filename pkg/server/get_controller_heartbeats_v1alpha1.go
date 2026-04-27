package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/heartbeat"
)

// heartbeatResponse aliases heartbeat.Heartbeat so that swag generates a
// clean OpenAPI / TypeScript type name (Heartbeat) for the value type used
// in the response below. The alias keeps the swag annotation co-located
// with the endpoint that defines the API contract, rather than in the
// underlying heartbeat package which knows nothing about HTTP responses.
type heartbeatResponse = heartbeat.Heartbeat // @name Heartbeat

// getControllerHeartbeatsResponse is the response body of GET
// /v1beta1/system/controller-heartbeats. Clients join Stages to heartbeats
// client-side using `stage.spec.controller` (or, when empty, the resolved
// `defaultControllerName`).
type getControllerHeartbeatsResponse struct {
	// Heartbeats is the most recent heartbeat from every controller that has
	// reported in indexed by controller name.
	Heartbeats map[string]heartbeatResponse `json:"heartbeats"`
	// DefaultController is the name of the default controller. The default
	// controller is often unnamed, so an empty string is a valid value. This is
	// included in the response to give clients a canonical identity to associate
	// with Stages that have no explicit `spec.controller`.
	DefaultController string `json:"defaultController"`
} // @name GetControllerHeartbeatsResponse

// @id GetControllerHeartbeats
// @Summary Get controller heartbeats
// @Description Get the most recent heartbeat from every controller that has
// @Description reported in. Any controller not represented in the response has
// @Description never reported a heartbeat and can therefore be assumed by the
// @Description caller to be dead or nonexistent.
// @Tags System
// @Security BearerAuth
// @Produce json
// @Success 200 {object} getControllerHeartbeatsResponse
// @Router /v1beta1/system/controller-heartbeats [get]
func (s *server) getControllerHeartbeats(c *gin.Context) {
	ctx := c.Request.Context()

	// The heartbeat records live in the Kargo namespace and may not be
	// readable by typical authenticated users. Use the internal (non-
	// authorizing) client: controller liveness is operational information that
	// should be visible to any authenticated user, and the data exposed
	// here (controller name, alive/dead, last-seen timestamp) is non-sensitive.
	heartbeats, err := heartbeat.GetAll(
		ctx, s.client.InternalClient(), s.cfg.KargoNamespace,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, getControllerHeartbeatsResponse{
		Heartbeats:        heartbeats,
		DefaultController: s.cfg.DefaultControllerName,
	})
}
