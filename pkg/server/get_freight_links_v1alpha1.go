package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/deeplinks"
)

type getFreightLinksResponse struct {
	Links  []deeplinks.ResolvedLink `json:"links"`
	Errors []string                 `json:"errors,omitempty"`
}

// @id GetFreightLinks
// @Summary Retrieve deep links for a Freight resource
// @Description Retrieve evaluated deep links for a Freight resource, combining
// @Description cluster-level links from ClusterConfig and project-level links
// @Description from ProjectConfig.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Success 200 {object} getFreightLinksResponse
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/links [get]
func (s *server) getFreightLinks(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	freight := s.getFreightByNameOrAliasForGin(c, project, nameOrAlias)
	if freight == nil {
		return
	}

	var linkDefs []kargoapi.DeepLink

	// We need to use the internal client here as regular/non-admin users may
	// not have GET permissions on cluster-configs.
	internalClient := s.client.InternalClient()

	clusterCfg := &kargoapi.ClusterConfig{}
	if err := internalClient.Get(ctx, client.ObjectKey{Name: api.ClusterConfigName}, clusterCfg); err != nil {
		if !apierrors.IsNotFound(err) {
			_ = c.Error(err)
			return
		}
	} else {
		linkDefs = append(linkDefs, clusterCfg.Spec.FreightLinks...)
	}

	projectCfg := &kargoapi.ProjectConfig{}
	if err := internalClient.Get(
		ctx, client.ObjectKey{Name: project, Namespace: project}, projectCfg,
	); err != nil {
		if !apierrors.IsNotFound(err) {
			_ = c.Error(err)
			return
		}
	} else {
		linkDefs = append(linkDefs, projectCfg.Spec.FreightLinks...)
	}

	if len(linkDefs) == 0 {
		c.JSON(http.StatusOK, getFreightLinksResponse{Links: []deeplinks.ResolvedLink{}})
		return
	}

	linkCtx, err := deeplinks.FreightContext(freight)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resolved, errs := deeplinks.EvaluateLinks(linkDefs, linkCtx)
	c.JSON(http.StatusOK, getFreightLinksResponse{Links: resolved, Errors: errs})
}
