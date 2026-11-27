package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/request"
)

// createFreightRequest is the request body for creating a piece of Freight.
type createFreightRequest struct {
	// Freight is the Freight resource to create. Its namespace is overridden with
	// the Project from the request path.
	Freight *kargoapi.Freight `json:"freight"`
	// Metadata is optional status metadata to record on the Freight. Each key is
	// set only if absent (fill-if-missing), so re-creating an already-existing
	// piece of Freight never overwrites metadata already recorded for it.
	Metadata map[string]apiextensionsv1.JSON `json:"metadata,omitempty"`
} // @name CreateFreightRequest

// @id CreateFreight
// @Summary Create a piece of Freight
// @Description Create a Freight resource, optionally recording status metadata on
// @Description it. The operation is idempotent: creating a piece of Freight that
// @Description already exists is not an error, and any metadata is applied
// @Description fill-if-missing so existing metadata is preserved.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body createFreightRequest true "Freight to create"
// @Success 201 {object} kargoapi.Freight "Freight was created"
// @Success 200 {object} kargoapi.Freight "Freight already existed"
// @Router /v1beta1/projects/{project}/freight [post]
func (s *server) createFreight(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	var req createFreightRequest
	if !request.BindJSON(c, &req) {
		return
	}
	if req.Freight == nil {
		_ = c.Error(libhttp.ErrorStr("freight is required", http.StatusBadRequest))
		return
	}
	if req.Freight.Name == "" {
		_ = c.Error(libhttp.ErrorStr("freight name is required", http.StatusBadRequest))
		return
	}

	freight := req.Freight.DeepCopy()
	// The Project from the path is authoritative; a Freight can only be created in
	// the Project the request is scoped to.
	freight.Namespace = project

	current, created, err := s.freightClientFn().CreateFreight(ctx, freight, req.Metadata)
	if err != nil {
		_ = c.Error(fmt.Errorf("create freight: %w", err))
		return
	}

	if created {
		c.JSON(http.StatusCreated, current)
		return
	}
	c.JSON(http.StatusOK, current)
}
