package kargomcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerStageTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "list_stages",
		Description:  "List stages in a Kargo project. Optionally filter by health status (Healthy, Unhealthy, Unknown).",
		OutputSchema: mustOutputSchema[stageListResult](),
		Annotations:  readOnly(),
	}, s.handleListStages)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_stage",
		Description:  "Get a single stage by name within a Kargo project.",
		OutputSchema: mustOutputSchema[stageResult](),
		Annotations:  readOnly(),
	}, s.handleGetStage)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "refresh_stage",
		Description: "Trigger an out-of-band refresh of a stage, causing it to " +
			"re-evaluate its subscriptions and re-sync its current state.",
		OutputSchema: mustOutputSchema[stageResult](),
		Annotations:  destructive(),
	}, s.handleRefreshStage)
}

// --- list_stages ---

type listStagesArgs struct {
	Project string `json:"project" jsonschema:"The name of the Kargo project"`
	Health  string `json:"health,omitempty" jsonschema:"Filter by health status: Healthy, Unhealthy, or Unknown"`
}

type stageListResult struct {
	Items []*stageResult `json:"items,omitempty"`
}

func (s *Server) handleListStages(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listStagesArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.ListStages(
		core.NewListStagesParams().WithProject(args.Project),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(filterStagesByHealth(res.Payload, args.Health))
}

// filterStagesByHealth optionally filters the stage list payload by health status.
// When health is empty the full list is returned unchanged.
func filterStagesByHealth(payload any, health string) any {
	if health == "" {
		return payload
	}
	want := strings.ToLower(health)
	data, err := json.Marshal(payload)
	if err != nil {
		return payload
	}
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return payload
	}
	var filtered []json.RawMessage
	for _, raw := range list.Items {
		var s struct {
			Status struct {
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		if strings.EqualFold(s.Status.Health.Status, want) {
			filtered = append(filtered, raw)
		}
	}
	if filtered == nil {
		filtered = []json.RawMessage{}
	}
	return map[string]any{"items": filtered}
}

// --- get_stage ---

type getStageArgs struct {
	Project string `json:"project" jsonschema:"The name of the Kargo project"`
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
	Name      string `json:"name,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Status    string `json:"status,omitempty"`
}

type stageCondition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type stageResult struct {
	Name             string                   `json:"name,omitempty"`
	Project          string                   `json:"namespace,omitempty"`
	CurrentFreight   []*stageFreightReference `json:"currentFreight,omitempty"`
	Health           *stageHealth             `json:"health,omitempty"`
	LastPromotion    *stageLastPromotion      `json:"lastPromotion,omitempty"`
	Conditions       []*stageCondition        `json:"conditions,omitempty"`
	Phase            string                   `json:"phase,omitempty"`
}

func (s *Server) handleGetStage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getStageArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetStage(
		core.NewGetStageParams().WithProject(args.Project).WithStage(args.Stage),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(res.Payload)
}

// --- refresh_stage ---

type refreshStageArgs struct {
	Project string `json:"project" jsonschema:"The name of the Kargo project"`
	Stage   string `json:"stage" jsonschema:"The name of the stage to refresh"`
}

func (s *Server) handleRefreshStage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args refreshStageArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	_, err = apiClient.Core.RefreshStage(
		core.NewRefreshStageParams().WithProject(args.Project).WithStage(args.Stage),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return okResult("Stage refresh triggered successfully.")
}
