package kargomcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
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
		Name:        "get_promotion_task",
		Description: "Get full details for a single PromotionTask in a Kargo project.",
		Annotations: readOnly(),
	}, s.handleGetPromotionTask)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_cluster_promotion_tasks",
		Description: "List cluster-scoped reusable PromotionTask templates " +
			"available across all projects.",
		OutputSchema: mustOutputSchema[struct {
			Items []promotionTaskSummary `json:"items"`
		}](),
		Annotations: readOnly(),
	}, s.handleListClusterPromotionTasks)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_cluster_promotion_task",
		Description: "Get full details for a single cluster-scoped PromotionTask.",
		Annotations: readOnly(),
	}, s.handleGetClusterPromotionTask)
}

type promotionTaskSummary struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps,omitempty"`
}

func promotionTaskToSummary(t *models.PromotionTask) promotionTaskSummary {
	s := promotionTaskSummary{}
	if t.Metadata != nil {
		s.Name = t.Metadata.Name
	}
	for _, step := range t.Spec.Steps {
		if step == nil {
			continue
		}
		label := step.Uses
		if step.As != "" {
			label = step.As + " (" + step.Uses + ")"
		}
		s.Steps = append(s.Steps, label)
	}
	return s
}

func clusterPromotionTaskToSummary(t *models.ClusterPromotionTask) promotionTaskSummary {
	s := promotionTaskSummary{}
	if t.Metadata != nil {
		s.Name = t.Metadata.Name
	}
	for _, step := range t.Spec.Steps {
		if step == nil {
			continue
		}
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
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
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
	summaries := make([]promotionTaskSummary, 0, len(res.Payload.Items))
	for _, t := range res.Payload.Items {
		if t != nil {
			summaries = append(summaries, promotionTaskToSummary(t))
		}
	}
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- get_promotion_task ---

type getPromotionTaskArgs struct {
	Project       string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	PromotionTask string `json:"promotion_task" jsonschema:"The name of the PromotionTask"`
}

func (s *Server) handleGetPromotionTask(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getPromotionTaskArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetPromotionTask(
		core.NewGetPromotionTaskParams().WithProject(project).WithPromotionTask(args.PromotionTask),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(toUnstructured(res.Payload)).Object)
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
	summaries := make([]promotionTaskSummary, 0, len(res.Payload.Items))
	for _, t := range res.Payload.Items {
		if t != nil {
			summaries = append(summaries, clusterPromotionTaskToSummary(t))
		}
	}
	return jsonAnyResult(map[string]any{"items": summaries})
}

// --- get_cluster_promotion_task ---

type getClusterPromotionTaskArgs struct {
	PromotionTask string `json:"promotion_task" jsonschema:"The name of the ClusterPromotionTask"`
}

func (s *Server) handleGetClusterPromotionTask(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getClusterPromotionTaskArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetClusterPromotionTask(
		core.NewGetClusterPromotionTaskParams().WithClusterPromotionTask(args.PromotionTask),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(toUnstructured(res.Payload)).Object)
}
