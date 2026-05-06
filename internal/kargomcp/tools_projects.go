package kargomcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

func (s *Server) registerProjectTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_projects",
		Description: "List all Kargo projects. Returns a compact summary per project.",
		OutputSchema: mustOutputSchema[struct {
			Items []projectSummary `json:"items"`
		}](),
		Annotations: readOnly(),
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

type projectSummary struct {
	Name              string `json:"name"`
	Ready             string `json:"ready,omitempty"`
	StageCount        int    `json:"stageCount,omitempty"`
	HealthyStages     int    `json:"healthyStages,omitempty"`
	WarehouseCount    int    `json:"warehouseCount,omitempty"`
	HealthyWarehouses int    `json:"healthyWarehouses,omitempty"`
}

func projectToSummary(p *models.Project) projectSummary {
	s := projectSummary{
		StageCount:        int(p.Status.Stats.Stages.Count),
		HealthyStages:     int(p.Status.Stats.Stages.Health.Healthy),
		WarehouseCount:    int(p.Status.Stats.Warehouses.Count),
		HealthyWarehouses: int(p.Status.Stats.Warehouses.Health.Healthy),
	}
	if p.Metadata != nil {
		s.Name = p.Metadata.Name
	}
	for _, c := range p.Status.Conditions {
		if c != nil && c.Type != nil && *c.Type == "Ready" {
			if c.Status != nil {
				s.Ready = *c.Status
			}
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
	summaries := make([]projectSummary, 0, len(res.Payload.Items))
	for _, p := range res.Payload.Items {
		if p != nil {
			summaries = append(summaries, projectToSummary(p))
		}
	}
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- get_project ---

type getProjectArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
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
	Name       string              `json:"name,omitempty"`
	Conditions []*projectCondition `json:"conditions,omitempty"`
	Stats      *projectStats       `json:"stats,omitempty"`
}

func (s *Server) handleGetProject(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getProjectArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetProject(
		core.NewGetProjectParams().WithProject(project),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(toUnstructured(res.Payload)).Object)
}
