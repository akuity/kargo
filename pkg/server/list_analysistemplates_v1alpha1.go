package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

func (s *server) ListAnalysisTemplates(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListAnalysisTemplatesRequest],
) (*connect.Response[svcv1alpha1.ListAnalysisTemplatesResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		// nolint:staticcheck
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("Argo Rollouts integration is not enabled"),
		)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list rolloutsapi.AnalysisTemplateList
	opts := []client.ListOption{
		client.InNamespace(project),
	}
	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list analysistemplates: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs rolloutsapi.AnalysisTemplate) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	ats := make([]*rolloutsapi.AnalysisTemplate, len(list.Items))
	for idx := range list.Items {
		ats[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListAnalysisTemplatesResponse{
		AnalysisTemplates: ats,
	}), nil
}

// nolint: lll
// @id ListAnalysisTemplates
// @Summary List AnalysisTemplates
// @Description List AnalysisTemplate resources from a project's namespace.
// @Tags Verifications, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} object "AnalysisTemplateList custom resource (github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1.AnalysisTemplateList)"
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
