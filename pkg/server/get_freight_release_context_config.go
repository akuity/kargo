package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id GetFreightReleaseContextConfig
// @Summary Retrieve effective release context configuration for Freight
// @Description Returns project annotation mappings when configured, otherwise cluster defaults.
// @Description Authorizes read access to the requested Freight before reading configuration internally.
// @Description Only annotation-key mappings are returned; Freight readers need not access full configuration resources.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Success 200 {object} kargoapi.ReleaseContextConfig
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/release-context-config [get]
func (s *server) getFreightReleaseContextConfig(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	if s.getFreightByNameOrAliasForGin(c, project, nameOrAlias) == nil {
		return
	}

	projectCfg, err := api.GetProjectConfig(ctx, s.client.InternalClient(), project)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if projectCfg != nil && projectCfg.Spec.ReleaseContext != nil {
		c.JSON(http.StatusOK, projectCfg.Spec.ReleaseContext)
		return
	}

	clusterCfg, err := api.GetClusterConfig(ctx, s.client.InternalClient())
	if err != nil {
		_ = c.Error(err)
		return
	}
	if clusterCfg != nil && clusterCfg.Spec.ReleaseContext != nil {
		c.JSON(http.StatusOK, clusterCfg.Spec.ReleaseContext)
		return
	}

	c.JSON(http.StatusOK, kargoapi.ReleaseContextConfig{})
}
