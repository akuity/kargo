package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
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
// @Param aliasOnly query bool false "Match only the Freight alias, not its name"
// @Success 200 {object} kargoapi.Freight "Freight custom resource (github.com/akuity/kargo/api/v1alpha1.Freight)"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias} [get]
func (s *server) getFreight(c *gin.Context) {
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	var freight *kargoapi.Freight
	var err error
	ctx := c.Request.Context()
	nameOnly := c.Query("nameOnly") == trueStr
	aliasOnly := c.Query("aliasOnly") == trueStr

	switch {
	case nameOnly && aliasOnly:
		_ = c.Error(libhttp.Error(
			errors.New("nameOnly and aliasOnly cannot be set at the same time"),
			http.StatusBadRequest,
		))
		return
	case nameOnly:
		freight, err = s.freightClient.GetFreight(ctx, project, nameOrAlias)
	case aliasOnly:
		freight, err = s.freightClient.GetFreightByAlias(ctx, project, nameOrAlias)
	default:
		freight, err = s.freightClient.GetFreightByNameOrAlias(ctx, project, nameOrAlias)
	}
	if err != nil {
		_ = c.Error(err)
		return
	}

	if freight == nil {
		_ = c.Error(libhttp.Error(
			errors.New("Freight not found"), // nolint: staticcheck
			http.StatusNotFound,
		))
		return 
	}

	c.JSON(http.StatusOK, freight)
}

// This keeps the kargoapi import above in scope for the @Success annotation
// on getFreight, which documents the response type without constructing one
// directly
var _ kargoapi.Freight
