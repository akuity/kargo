package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/request"
)

// patchFreightMetadataRequest is the request body for updating a piece of
// Freight's status metadata.
type patchFreightMetadataRequest struct {
	// Metadata is the status metadata to record on the Freight. Each key is
	// upserted: an existing value under the same key is overwritten, and keys
	// not named in the request are left as they are.
	Metadata map[string]apiextensionsv1.JSON `json:"metadata"`
} // @name PatchFreightMetadataRequest

// @id PatchFreightMetadata
// @Summary Update a Freight resource's status metadata
// @Description Upsert keys in the status metadata of a Freight resource. A value
// @Description already recorded under a key named in the request is
// @Description overwritten; other keys are left as they are.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param freight path string true "Freight name"
// @Param body body patchFreightMetadataRequest true "Metadata"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/freight/{freight}/metadata [patch]
func (s *server) patchFreightMetadata(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightName := c.Param("freight")

	var req patchFreightMetadataRequest
	if !request.BindJSON(c, &req) {
		return
	}
	if len(req.Metadata) == 0 {
		_ = c.Error(libhttp.ErrorStr("metadata is required", http.StatusBadRequest))
		return
	}
	if _, ok := req.Metadata[""]; ok {
		_ = c.Error(libhttp.ErrorStr("metadata keys must not be empty", http.StatusBadRequest))
		return
	}

	err := s.freightClientFn().PatchFreightMetadata(ctx, project, freightName, req.Metadata)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
