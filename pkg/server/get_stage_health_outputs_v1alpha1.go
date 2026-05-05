package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// maxStageHealthOutputsBatch is a soft limit on the number of Stage names a
// single GetStageHealthOutputs call may request. Chosen to comfortably
// exceed the largest known project scale while still bounding worst-case
// response size.
const maxStageHealthOutputsBatch = 1000

// GetStageHealthOutputs returns the raw health output blob for the specified
// Stages in a project. Stages that do not exist or have no health output
// recorded are omitted from the response map. Intended for clients that use
// ListStages with summary=true for the list and lazily resolve per-argocd-app
// health for Stages currently in viewport.
func (s *server) GetStageHealthOutputs(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetStageHealthOutputsRequest],
) (*connect.Response[svcv1alpha1.GetStageHealthOutputsResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	stageNames := req.Msg.GetStageNames()
	if len(stageNames) > maxStageHealthOutputsBatch {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"stage_names exceeds maximum batch size of %d (got %d)",
				maxStageHealthOutputsBatch, len(stageNames),
			),
		)
	}

	outputs, err := api.ListStageHealthOutputs(ctx, s.client, project, stageNames)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&svcv1alpha1.GetStageHealthOutputsResponse{
		HealthOutputs: outputs,
	}), nil
}

// @id GetStageHealthOutputs
// @Summary Get Stage Health Outputs
// @Description Return the raw health output blob for the specified Stages
// @Description in a project. Stages that do not exist or have no recorded
// @Description health output are omitted from the response. Intended for
// @Description clients that use ListStages with summary=true (which omits the output
// @Description blob) and need to lazily resolve health for Stages currently
// @Description in viewport.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stageNames query []string true "Stage names to fetch health outputs for" collectionFormat(multi)
// @Success 200 {object} svcv1alpha1.GetStageHealthOutputsResponse
// @Router /v1beta1/projects/{project}/stage-health-outputs [get]
func (s *server) getStageHealthOutputs(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	stageNames := c.QueryArray("stageNames")
	if len(stageNames) > maxStageHealthOutputsBatch {
		_ = c.Error(libhttp.Error(
			fmt.Errorf(
				"stageNames exceeds maximum batch size of %d (got %d)",
				maxStageHealthOutputsBatch, len(stageNames),
			),
			http.StatusBadRequest,
		))
		return
	}

	outputs, err := api.ListStageHealthOutputs(ctx, s.client, project, stageNames)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &svcv1alpha1.GetStageHealthOutputsResponse{
		HealthOutputs: outputs,
	})
}
