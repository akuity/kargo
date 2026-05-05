package kargomcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerWarehouseTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_warehouses",
		Description: "List warehouses in a Kargo project. Returns a compact summary per warehouse.",
		OutputSchema: mustOutputSchema[struct {
			Items []warehouseSummary `json:"items"`
		}](),
		Annotations: readOnly(),
	}, s.handleListWarehouses)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_warehouse",
		Description:  "Get full details for a single warehouse.",
		OutputSchema: mustOutputSchema[warehouseResult](),
		Annotations:  readOnly(),
	}, s.handleGetWarehouse)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "refresh_warehouse",
		Description: "Trigger an out-of-band refresh of a warehouse, causing it to " +
			"re-check its artifact sources and produce new Freight if new artifacts are found.",
		Annotations: destructive(),
	}, s.handleRefreshWarehouse)
}

// --- list_warehouses ---

type listWarehousesArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
}

type warehouseJSON struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Status struct {
		Conditions []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"conditions"`
		DiscoveredArtifacts *struct {
			DiscoveredAt string `json:"discoveredAt"`
		} `json:"discoveredArtifacts"`
	} `json:"status"`
}

type warehouseSummary struct {
	Name                    string `json:"name"`
	Ready                   string `json:"ready,omitempty"`
	Healthy                 string `json:"healthy,omitempty"`
	LastFreightDiscoveredAt string `json:"lastFreightDiscoveredAt,omitempty"`
}

func warehouseToSummary(w warehouseJSON) warehouseSummary {
	s := warehouseSummary{Name: w.Metadata.Name}
	for _, c := range w.Status.Conditions {
		switch c.Type {
		case "Ready":
			s.Ready = c.Status
		case "Healthy":
			s.Healthy = c.Status
		}
	}
	if w.Status.DiscoveredArtifacts != nil {
		s.LastFreightDiscoveredAt = w.Status.DiscoveredArtifacts.DiscoveredAt
	}
	return s
}

func (s *Server) handleListWarehouses(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listWarehousesArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.ListWarehouses(
		core.NewListWarehousesParams().WithProject(project),
		nil,
	)
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
	summaries := make([]warehouseSummary, 0, len(list.Items))
	for _, raw := range list.Items {
		var w warehouseJSON
		if err := json.Unmarshal(raw, &w); err != nil {
			continue
		}
		summaries = append(summaries, warehouseToSummary(w))
	}
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- get_warehouse ---

type getWarehouseArgs struct {
	Project   string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Warehouse string `json:"warehouse" jsonschema:"The name of the warehouse"`
}

type warehouseCondition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type warehouseResult struct {
	Name       string                `json:"name,omitempty"`
	Project    string                `json:"namespace,omitempty"`
	Conditions []*warehouseCondition `json:"conditions,omitempty"`
}

func (s *Server) handleGetWarehouse(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getWarehouseArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetWarehouse(
		core.NewGetWarehouseParams().WithProject(project).WithWarehouse(args.Warehouse),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(res.Payload))
}

// --- refresh_warehouse ---

type refreshWarehouseArgs struct {
	Project   string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Warehouse string `json:"warehouse" jsonschema:"The name of the warehouse to refresh"`
}

func (s *Server) handleRefreshWarehouse(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args refreshWarehouseArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	_, err = apiClient.Core.RefreshWarehouse(
		core.NewRefreshWarehouseParams().WithProject(project).WithWarehouse(args.Warehouse),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return okResult("Warehouse refresh triggered successfully.")
}
