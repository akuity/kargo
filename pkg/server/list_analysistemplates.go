package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

// nolint: lll
// @id ListAnalysisTemplates
// @Summary List AnalysisTemplates
// @Description List AnalysisTemplate resources from a project's namespace.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} rolloutsapi.AnalysisTemplateList "AnalysisTemplateList custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.AnalysisTemplateList)"
// @Router /v1beta1/projects/{project}/analysis-templates [get]
func (s *server) listAnalysisTemplates(c *gin.Context) {
	if !s.cfg.RolloutsIntegrationEnabled {
		_ = c.Error(errArgoRolloutsIntegrationDisabled)
		return
	}

	ctx := c.Request.Context()

	project := c.Param("project")

	list := &rolloutsapi.AnalysisTemplateList{}
	if err := s.client.List(
		ctx, list, client.InNamespace(project),
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs rolloutsapi.AnalysisTemplate) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}
