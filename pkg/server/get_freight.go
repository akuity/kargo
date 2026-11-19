package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id GetFreight
// @Summary Retrieve a Freight resource
// @Description Retrieve a Freight resource from a project's namespace by name
// @Description or alias.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param nameOnly query bool false "Match only the Freight name, not its alias"
// @Success 200 {object} kargoapi.Freight "Freight custom resource (github.com/akuity/kargo/api/v1alpha1.Freight)"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias} [get]
func (s *server) getFreight(c *gin.Context) {
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	var freight *kargoapi.Freight
	var err error
	ctx := c.Request.Context()
	if c.Query("nameOnly") == trueStr {
		freight, err = s.getFreightByName(ctx, project, nameOrAlias)
	} else {
		freight, err = s.getFreightByNameOrAlias(ctx, project, nameOrAlias)
	}
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, freight)
}

// This keeps the kargoapi import above in scope for the @Success annotation
// on getFreight, which documents the response type without constructing one
// directly (the actual response is produced by the shared
// getFreightByNameOrAlias helper).
var _ kargoapi.Freight
