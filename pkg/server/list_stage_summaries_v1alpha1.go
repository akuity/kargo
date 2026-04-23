package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// ListStageSummaries returns a lightweight projection of the Stages in the
// given project. When the request's FreightOrigins field is non-empty, only
// Stages that subscribe to Freight from at least one of the named Warehouses
// are returned.
func (s *server) ListStageSummaries(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStageSummariesRequest],
) (*connect.Response[svcv1alpha1.ListStageSummariesResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	filtered := filterStagesByWarehouses(list.Items, req.Msg.GetFreightOrigins())

	slices.SortFunc(filtered, func(a, b kargoapi.Stage) int {
		return strings.Compare(a.Name, b.Name)
	})

	summaries := make([]*svcv1alpha1.StageSummary, len(filtered))
	for i := range filtered {
		summaries[i] = stageToSummary(&filtered[i])
	}

	return connect.NewResponse(&svcv1alpha1.ListStageSummariesResponse{
		StageSummaries:  summaries,
		ResourceVersion: list.ResourceVersion,
	}), nil
}

// filterStagesByWarehouses returns the subset of stages that request Freight
// from at least one of the named Warehouses. Both direct and indirect
// subscribers match: membership is determined solely by FreightRequest.Origin,
// independent of whether Sources.Direct is true or the Freight arrives via an
// upstream Stage. An empty warehouseNames slice returns all stages unchanged.
func filterStagesByWarehouses(
	stages []kargoapi.Stage,
	warehouseNames []string,
) []kargoapi.Stage {
	if len(warehouseNames) == 0 {
		return stages
	}
	want := make(map[string]struct{}, len(warehouseNames))
	for _, n := range warehouseNames {
		if n == "" {
			continue
		}
		want[n] = struct{}{}
	}
	if len(want) == 0 {
		return stages
	}
	out := stages[:0:0]
	for i := range stages {
		if stageMatchesAnyWarehouse(&stages[i], want) {
			out = append(out, stages[i])
		}
	}
	return out
}

// stageMatchesAnyWarehouse returns true if the given Stage requests Freight
// from any of the Warehouse names in the given set.
func stageMatchesAnyWarehouse(
	stage *kargoapi.Stage,
	warehouseNames map[string]struct{},
) bool {
	for _, req := range stage.Spec.RequestedFreight {
		if _, ok := warehouseNames[req.Origin.Name]; ok {
			return true
		}
	}
	return false
}

// @id ListStageSummaries
// @Summary List Stage Summaries
// @Description List a lightweight projection of Stage resources from a
// @Description project's namespace. Intended for UI list and graph views that
// @Description need metadata and current state for many Stages at once but do
// @Description not need full FreightHistory, PromotionTemplate steps, or
// @Description Verification configuration. Use GetStage for detail fields.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freightOrigins query []string false "Warehouse name(s) to filter by" collectionFormat(multi)
// @Success 200 {object} svcv1alpha1.ListStageSummariesResponse
// @Router /v1beta1/projects/{project}/stage-summaries [get]
func (s *server) listStageSummaries(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchStageSummaries(c, project)
		return
	}

	freightOrigins := c.QueryArray("freightOrigins")

	list := &kargoapi.StageList{}
	if err := s.client.List(ctx, list, client.InNamespace(project)); err != nil {
		_ = c.Error(err)
		return
	}

	filtered := filterStagesByWarehouses(list.Items, freightOrigins)

	slices.SortFunc(filtered, func(a, b kargoapi.Stage) int {
		return strings.Compare(a.Name, b.Name)
	})

	summaries := make([]*svcv1alpha1.StageSummary, len(filtered))
	for i := range filtered {
		summaries[i] = stageToSummary(&filtered[i])
	}

	c.JSON(http.StatusOK, &svcv1alpha1.ListStageSummariesResponse{
		StageSummaries:  summaries,
		ResourceVersion: list.ResourceVersion,
	})
}

func (s *server) watchStageSummaries(c *gin.Context, project string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	freightOrigins := c.QueryArray("freightOrigins")
	want := warehouseNameSet(freightOrigins)

	opts := []client.ListOption{client.InNamespace(project)}
	if rv := c.Query("resourceVersion"); rv != "" {
		opts = append(opts, &client.ListOptions{
			Raw: &metav1.ListOptions{ResourceVersion: rv},
		})
	}

	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, opts...)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch stage summaries: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	setSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !writeSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}
			stage, ok := convertWatchEventObject(c, e, (*kargoapi.Stage)(nil))
			if !ok {
				continue
			}
			if len(want) > 0 && !stageMatchesAnyWarehouse(stage, want) {
				continue
			}
			if !sendSSEWatchEvent(c, e.Type, stageToSummary(stage)) {
				return
			}
		}
	}
}

// warehouseNameSet builds a set from the given names, skipping empty entries.
// Returns nil if no non-empty names are supplied, signaling "no filter".
func warehouseNameSet(names []string) map[string]struct{} {
	if len(names) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		want[n] = struct{}{}
	}
	if len(want) == 0 {
		return nil
	}
	return want
}
