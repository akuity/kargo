package kargomcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerProjectTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "list_projects",
		Description:  "List all Kargo projects the authenticated user has access to.",
		OutputSchema: mustOutputSchema[projectListResult](),
		Annotations:  readOnly(),
	}, s.handleListProjects)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_project",
		Description:  "Get a single Kargo project by name.",
		OutputSchema: mustOutputSchema[projectResult](),
		Annotations:  readOnly(),
	}, s.handleGetProject)
}

// --- list_projects ---

type listProjectsArgs struct{}

type projectListResult struct {
	Items []*projectResult `json:"items,omitempty"`
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
	return jsonAnyResult(res.Payload)
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
	Count  int64             `json:"count,omitempty"`
	Health map[string]int64  `json:"health,omitempty"`
}

type warehouseStats struct {
	Count  int64             `json:"count,omitempty"`
	Health map[string]int64  `json:"health,omitempty"`
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
	return jsonAnyResult(res.Payload)
}
