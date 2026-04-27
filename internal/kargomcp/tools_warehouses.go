package kargomcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerWarehouseTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "list_warehouses",
		Description:  "List all warehouses in a Kargo project.",
		OutputSchema: mustOutputSchema[warehouseListResult](),
		Annotations:  readOnly(),
	}, s.handleListWarehouses)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_warehouse",
		Description:  "Get a single warehouse by name within a Kargo project.",
		OutputSchema: mustOutputSchema[warehouseResult](),
		Annotations:  readOnly(),
	}, s.handleGetWarehouse)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "refresh_warehouse",
		Description: "Trigger an out-of-band refresh of a warehouse, causing it to " +
			"re-check its artifact sources and produce new Freight if new artifacts are found.",
		OutputSchema: mustOutputSchema[warehouseResult](),
		Annotations:  destructive(),
	}, s.handleRefreshWarehouse)
}

// --- list_warehouses ---

type listWarehousesArgs struct {
	Project string `json:"project" jsonschema:"The name of the Kargo project"`
}

type warehouseListResult struct {
	Items []*warehouseResult `json:"items,omitempty"`
}

func (s *Server) handleListWarehouses(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listWarehousesArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.ListWarehouses(
		core.NewListWarehousesParams().WithProject(args.Project),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(res.Payload)
}

// --- get_warehouse ---

type getWarehouseArgs struct {
	Project   string `json:"project" jsonschema:"The name of the Kargo project"`
	Warehouse string `json:"warehouse" jsonschema:"The name of the warehouse"`
}

type warehouseCondition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type warehouseResult struct {
	Name       string               `json:"name,omitempty"`
	Project    string               `json:"namespace,omitempty"`
	Conditions []*warehouseCondition `json:"conditions,omitempty"`
}

func (s *Server) handleGetWarehouse(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getWarehouseArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetWarehouse(
		core.NewGetWarehouseParams().WithProject(args.Project).WithWarehouse(args.Warehouse),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(res.Payload)
}

// --- refresh_warehouse ---

type refreshWarehouseArgs struct {
	Project   string `json:"project" jsonschema:"The name of the Kargo project"`
	Warehouse string `json:"warehouse" jsonschema:"The name of the warehouse to refresh"`
}

func (s *Server) handleRefreshWarehouse(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args refreshWarehouseArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	_, err = apiClient.Core.RefreshWarehouse(
		core.NewRefreshWarehouseParams().WithProject(args.Project).WithWarehouse(args.Warehouse),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return okResult("Warehouse refresh triggered successfully.")
}
