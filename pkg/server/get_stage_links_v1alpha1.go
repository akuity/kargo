package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/deeplinks"
	libhttp "github.com/akuity/kargo/pkg/http"
)

type getStageLinksResponse struct {
	Links  []deeplinks.ResolvedLink `json:"links"`
	Errors []string                 `json:"errors,omitempty"`
}

// @id GetStageLinks
// @Summary Retrieve deep links for a Stage resource
// @Description Retrieve evaluated deep links for a Stage resource, combining
// @Description cluster-level links from ClusterConfig and project-level links
// @Description from ProjectConfig.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 {object} getStageLinksResponse
// @Router /v1beta1/projects/{project}/stages/{stage}/links [get]
func (s *server) getStageLinks(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	stage := &kargoapi.Stage{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: stageName, Namespace: project}, stage,
	); err != nil {
		if apierrors.IsNotFound(err) {
			_ = c.Error(libhttp.ErrorStr(
				fmt.Sprintf("Stage %q not found in project %q", stageName, project),
				http.StatusNotFound,
			))
			return
		}
		_ = c.Error(err)
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
		linkDefs = append(linkDefs, clusterCfg.Spec.StageLinks...)
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
		linkDefs = append(linkDefs, projectCfg.Spec.StageLinks...)
	}

	if len(linkDefs) == 0 {
		c.JSON(http.StatusOK, getStageLinksResponse{Links: []deeplinks.ResolvedLink{}})
		return
	}

	linkCtx, err := deeplinks.StageContext(stage)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resolved, errs := deeplinks.EvaluateLinks(linkDefs, linkCtx)
	c.JSON(http.StatusOK, getStageLinksResponse{Links: resolved, Errors: errs})
}
