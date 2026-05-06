package kargomcp

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

func (s *Server) registerStageTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_stages",
		Description: "List stages in a Kargo project. Returns a compact summary per stage. " +
			"Optionally filter by warehouse (stages that request freight from those warehouses) " +
			"and/or health status (Healthy, Unhealthy, Unknown).",
		OutputSchema: mustOutputSchema[struct {
			Items []stageSummary `json:"items"`
		}](),
		Annotations: readOnly(),
	}, s.handleListStages)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "get_stage",
		Description: "Get full details for a single stage. " +
			"Note: status.freightHistory is omitted — use get_stage_freight_history for that data.",
		OutputSchema: mustOutputSchema[stageResult](),
		Annotations:  readOnly(),
	}, s.handleGetStage)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "get_stage_freight_history",
		Description: "Get the freight history for a stage — the sequence of freight " +
			"collections that have been promoted through it, with verification results. " +
			"Use this when you need to audit what freight has passed through a stage " +
			"or investigate verification failures.",
		Annotations: readOnly(),
	}, s.handleGetStageFreightHistory)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "refresh_stage",
		Description: "Trigger an out-of-band refresh of a stage, causing it to " +
			"re-evaluate its subscriptions and re-sync its current state.",
		Annotations: destructive(),
	}, s.handleRefreshStage)
}

// --- list_stages ---

type listStagesArgs struct {
	Project    string   `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Warehouses []string `json:"warehouses,omitempty" jsonschema:"Filter to stages that request freight from these warehouses"`                    //nolint:lll
	Health     string   `json:"health,omitempty" jsonschema:"Filter by health status: Healthy, Unhealthy, or Unknown"`
}

type stageSummary struct {
	Name                 string              `json:"name"`
	Health               string              `json:"health,omitempty"`
	HealthIssues         []string            `json:"healthIssues,omitempty"`
	CurrentFreight       string              `json:"currentFreight,omitempty"`
	AutoPromotionEnabled bool                `json:"autoPromotionEnabled,omitempty"`
	CurrentPromotion     string              `json:"currentPromotion,omitempty"`
	LastPromotion        *lastPromotionBrief `json:"lastPromotion,omitempty"`
}

type lastPromotionBrief struct {
	Name       string `json:"name,omitempty"`
	Phase      string `json:"phase,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Message    string `json:"message,omitempty"`
}

func stageToSummary(st *models.Stage) stageSummary {
	s := stageSummary{
		Health:               st.Status.Health.Status,
		HealthIssues:         st.Status.Health.Issues,
		CurrentFreight:       st.Status.FreightSummary,
		AutoPromotionEnabled: st.Status.AutoPromotionEnabled,
		CurrentPromotion:     st.Status.CurrentPromotion.Name,
	}
	if st.Metadata != nil {
		s.Name = st.Metadata.Name
	}
	if st.Status.LastPromotion.Name != "" {
		s.LastPromotion = &lastPromotionBrief{
			Name:       st.Status.LastPromotion.Name,
			Phase:      st.Status.LastPromotion.Status.Phase,
			FinishedAt: st.Status.LastPromotion.FinishedAt,
			Message:    st.Status.LastPromotion.Status.Message,
		}
	}
	return s
}

func (s *Server) handleListStages(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listStagesArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	params := core.NewListStagesParams().WithProject(project)
	if len(args.Warehouses) > 0 {
		params = params.WithFreightOrigins(args.Warehouses)
	}
	res, err := apiClient.Core.ListStages(params, nil)
	if err != nil {
		return errResult(err)
	}
	want := strings.ToLower(args.Health)
	summaries := make([]stageSummary, 0, len(res.Payload.Items))
	for _, st := range res.Payload.Items {
		if st == nil {
			continue
		}
		if want != "" && !strings.EqualFold(st.Status.Health.Status, want) {
			continue
		}
		summaries = append(summaries, stageToSummary(st))
	}
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- get_stage ---

type getStageArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Stage   string `json:"stage" jsonschema:"The name of the stage"`
}

type stageFreightReference struct {
	Name   string `json:"name,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Origin string `json:"origin,omitempty"`
}

type stageHealth struct {
	Status string   `json:"status,omitempty"`
	Issues []string `json:"issues,omitempty"`
}

type stageLastPromotion struct {
	Name       string `json:"name,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Status     string `json:"status,omitempty"`
}

type stageCondition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type stageResult struct {
	Name           string                   `json:"name,omitempty"`
	Project        string                   `json:"namespace,omitempty"`
	CurrentFreight []*stageFreightReference `json:"currentFreight,omitempty"`
	Health         *stageHealth             `json:"health,omitempty"`
	LastPromotion  *stageLastPromotion      `json:"lastPromotion,omitempty"`
	Conditions     []*stageCondition        `json:"conditions,omitempty"`
	Phase          string                   `json:"phase,omitempty"`
}

func (s *Server) handleGetStage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getStageArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetStage(
		core.NewGetStageParams().WithProject(project).WithStage(args.Stage),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	u := toUnstructured(res.Payload)
	sanitizeResource(u)
	if status, ok := u.Object["status"].(map[string]any); ok {
		delete(status, "freightHistory")
	}
	return jsonAnyResult(u.Object)
}

// --- get_stage_freight_history ---

type getStageFreightHistoryArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Stage   string `json:"stage" jsonschema:"The name of the stage"`
}

func (s *Server) handleGetStageFreightHistory(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getStageFreightHistoryArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetStage(
		core.NewGetStageParams().WithProject(project).WithStage(args.Stage),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	u := toUnstructured(res.Payload)
	status, _ := u.Object["status"].(map[string]any)
	history := status["freightHistory"]
	if history == nil {
		history = []any{}
	}
	return jsonAnyResult(map[string]any{"items": history})
}

// --- refresh_stage ---

type refreshStageArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Stage   string `json:"stage" jsonschema:"The name of the stage to refresh"`
}

func (s *Server) handleRefreshStage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args refreshStageArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	_, err = apiClient.Core.RefreshStage(
		core.NewRefreshStageParams().WithProject(project).WithStage(args.Stage),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return okResult("Stage refresh triggered successfully.")
}
