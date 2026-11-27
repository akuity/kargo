package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id PatchFreightAlias
// @Summary Patch a Freight resource's alias
// @Description Patch a Freight resource's human-friendly alias.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param newAlias query string true "New alias"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/alias [patch]
func (s *server) patchFreightAliasHandler(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")
	newAlias := c.Query("newAlias")

	if newAlias == "" {
		_ = c.Error(libhttp.ErrorStr(
			"newAlias query parameter is required",
			http.StatusBadRequest,
		))
		return
	}

	err := s.freightClientFn().PatchFreightAlias(ctx, project, nameOrAlias, newAlias)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusOK)
}
