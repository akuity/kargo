package kargomcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerPromotionTaskTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_promotion_tasks",
		Description: "List reusable PromotionTask templates in a Kargo project. " +
			"PromotionTasks are named, reusable sequences of promotion steps " +
			"that stages can reference in their promotionTemplate.",
		OutputSchema: mustOutputSchema[struct {
			Items []promotionTaskSummary `json:"items"`
		}](),
		Annotations: readOnly(),
	}, s.handleListPromotionTasks)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_cluster_promotion_tasks",
		Description: "List cluster-scoped reusable PromotionTask templates " +
			"available across all projects.",
		OutputSchema: mustOutputSchema[struct {
			Items []promotionTaskSummary `json:"items"`
		}](),
		Annotations: readOnly(),
	}, s.handleListClusterPromotionTasks)
}

// promotionTaskJSON is the intake struct for summary projection.
type promotionTaskJSON struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Steps []struct {
			Uses string `json:"uses"`
			As   string `json:"as,omitempty"`
		} `json:"steps"`
	} `json:"spec"`
}

type promotionTaskSummary struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps,omitempty"`
}

func promotionTaskToSummary(t promotionTaskJSON) promotionTaskSummary {
	s := promotionTaskSummary{Name: t.Metadata.Name}
	for _, step := range t.Spec.Steps {
		label := step.Uses
		if step.As != "" {
			label = step.As + " (" + step.Uses + ")"
		}
		s.Steps = append(s.Steps, label)
	}
	return s
}

// --- list_promotion_tasks ---

type listPromotionTasksArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"`
}

func (s *Server) handleListPromotionTasks(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listPromotionTasksArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.ListPromotionTasks(
		core.NewListPromotionTasksParams().WithProject(project),
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
	summaries := projectItems(list.Items, promotionTaskToSummary)
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- list_cluster_promotion_tasks ---

type listClusterPromotionTasksArgs struct{}

func (s *Server) handleListClusterPromotionTasks(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ listClusterPromotionTasksArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.ListClusterPromotionTasks(
		core.NewListClusterPromotionTasksParams(),
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
	summaries := projectItems(list.Items, promotionTaskToSummary)
	return jsonAnyResult(map[string]any{"items": summaries})
}
