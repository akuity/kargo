package kargomcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerProjectTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "list_projects",
		Description:  "List all Kargo projects. Returns a compact summary per project.",
		OutputSchema: mustOutputSchema[struct {
			Items []projectSummary `json:"items"`
		}](),
		Annotations:  readOnly(),
	}, s.handleListProjects)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_project",
		Description:  "Get full details for a single Kargo project.",
		OutputSchema: mustOutputSchema[projectResult](),
		Annotations:  readOnly(),
	}, s.handleGetProject)
}

// --- list_projects ---

type listProjectsArgs struct{}

// projectJSON is the intake struct for decoding list items — only the fields we
// want to surface in the summary.
type projectJSON struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Status struct {
		Conditions []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"conditions"`
		Stats struct {
			Stages struct {
				Count  int            `json:"count"`
				Health map[string]int `json:"health"`
			} `json:"stages"`
			Warehouses struct {
				Count  int            `json:"count"`
				Health map[string]int `json:"health"`
			} `json:"warehouses"`
		} `json:"stats"`
	} `json:"status"`
}

type projectSummary struct {
	Name              string `json:"name"`
	Ready             string `json:"ready,omitempty"`
	StageCount        int    `json:"stageCount,omitempty"`
	HealthyStages     int    `json:"healthyStages,omitempty"`
	WarehouseCount    int    `json:"warehouseCount,omitempty"`
	HealthyWarehouses int    `json:"healthyWarehouses,omitempty"`
}

func projectToSummary(p projectJSON) projectSummary {
	s := projectSummary{
		Name:              p.Metadata.Name,
		StageCount:        p.Status.Stats.Stages.Count,
		HealthyStages:     p.Status.Stats.Stages.Health["healthy"],
		WarehouseCount:    p.Status.Stats.Warehouses.Count,
		HealthyWarehouses: p.Status.Stats.Warehouses.Health["healthy"],
	}
	for _, c := range p.Status.Conditions {
		if c.Type == "Ready" {
			s.Ready = c.Status
			break
		}
	}
	return s
}

func (s *Server) handleListProjects(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ listProjectsArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.ListProjects(core.NewListProjectsParams(), nil)
	if err != nil {
		return errResult(err)
	}
	data, _ := json.Marshal(res.Payload)
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return errResult(err)
	}
	summaries := projectItems[projectJSON, projectSummary](list.Items, projectToSummary)
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- get_project ---

type getProjectArgs struct {
	Project string `json:"project" jsonschema:"The name of the Kargo project"`
}

type projectCondition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type projectStats struct {
	Stages     *stageStats     `json:"stages,omitempty"`
	Warehouses *warehouseStats `json:"warehouses,omitempty"`
}

type stageStats struct {
	Count  int64            `json:"count,omitempty"`
	Health map[string]int64 `json:"health,omitempty"`
}

type warehouseStats struct {
	Count  int64            `json:"count,omitempty"`
	Health map[string]int64 `json:"health,omitempty"`
}

type projectResult struct {
	Name       string             `json:"name,omitempty"`
	Conditions []*projectCondition `json:"conditions,omitempty"`
	Stats      *projectStats      `json:"stats,omitempty"`
}

func (s *Server) handleGetProject(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getProjectArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetProject(
		core.NewGetProjectParams().WithProject(args.Project),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(res.Payload))
}

