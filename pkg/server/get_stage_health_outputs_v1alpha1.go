package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// maxStageHealthOutputsBatch is a soft limit on the number of Stage names a
// single GetStageHealthOutputs call may request. Chosen to comfortably
// exceed the largest known project scale while still bounding worst-case
// response size.
const maxStageHealthOutputsBatch = 1000

// GetStageHealthOutputs returns the raw health output blob for the specified
// Stages in a project. Stages that do not exist or have no health output
// recorded are omitted from the response map. Intended for clients that use
// ListStageSummaries for the list and lazily resolve per-argocd-app health
// for Stages currently in viewport.
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

	wanted := uniqueNonEmptyStrings(req.Msg.GetStageNames())
	if len(wanted) == 0 {
		return connect.NewResponse(&svcv1alpha1.GetStageHealthOutputsResponse{}), nil
	}
	if len(wanted) > maxStageHealthOutputsBatch {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf(
				"stage_names exceeds maximum batch size of %d (got %d)",
				maxStageHealthOutputsBatch, len(wanted),
			),
		)
	}

	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	outputs := make(map[string][]byte, len(wanted))
	for i := range list.Items {
		st := &list.Items[i]
		if _, want := wanted[st.Name]; !want {
			continue
		}
		if st.Status.Health == nil || st.Status.Health.Output == nil {
			continue
		}
		outputs[st.Name] = st.Status.Health.Output.Raw
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
// @Description clients that use ListStageSummaries (which omits the output
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

	wanted := uniqueNonEmptyStrings(c.QueryArray("stageNames"))
	if len(wanted) == 0 {
		c.JSON(http.StatusOK, &svcv1alpha1.GetStageHealthOutputsResponse{})
		return
	}
	if len(wanted) > maxStageHealthOutputsBatch {
		_ = c.Error(fmt.Errorf(
			"stageNames exceeds maximum batch size of %d (got %d)",
			maxStageHealthOutputsBatch, len(wanted),
		))
		return
	}

	list := &kargoapi.StageList{}
	if err := s.client.List(ctx, list, client.InNamespace(project)); err != nil {
		_ = c.Error(err)
		return
	}

	outputs := make(map[string][]byte, len(wanted))
	for i := range list.Items {
		st := &list.Items[i]
		if _, want := wanted[st.Name]; !want {
			continue
		}
		if st.Status.Health == nil || st.Status.Health.Output == nil {
			continue
		}
		outputs[st.Name] = st.Status.Health.Output.Raw
	}

	c.JSON(http.StatusOK, &svcv1alpha1.GetStageHealthOutputsResponse{
		HealthOutputs: outputs,
	})
}

// uniqueNonEmptyStrings returns a set of the non-empty entries in names,
// deduplicated. Returns an empty (non-nil) map if no non-empty entries are
// present.
func uniqueNonEmptyStrings(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		set[n] = struct{}{}
	}
	return set
}
