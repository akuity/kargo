package kargomcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

func (s *Server) registerPromotionTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_promotions",
		Description: "List promotions in a Kargo project, newest first. Returns a compact summary per promotion. " +
			"Optionally filter by stage and/or phase (Running, Succeeded, Failed, Errored, Pending, Aborted).",
		OutputSchema: mustOutputSchema[struct {
			Items []promotionSummary `json:"items"`
		}](),
		Annotations: readOnly(),
	}, s.handleListPromotions)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_promotion",
		Description:  "Get a single promotion by name within a Kargo project.",
		OutputSchema: mustOutputSchema[promotionResult](),
		Annotations:  readOnly(),
	}, s.handleGetPromotion)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "promote_to_stage",
		Description: "Promote a piece of freight to a specific stage. " +
			"Provide either freight (name) or freight_alias, not both.",
		OutputSchema: mustOutputSchema[promotionResult](),
		Annotations:  destructive(),
	}, s.handlePromoteToStage)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "promote_downstream",
		Description: "Promote a piece of freight to all stages immediately downstream of a given stage. " +
			"Provide either freight (name) or freight_alias, not both.",
		OutputSchema: mustOutputSchema[struct {
			Items []promotionSummary `json:"items"`
		}](),
		Annotations: destructive(),
	}, s.handlePromoteDownstream)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "abort_promotion",
		Description:  "Abort a non-terminal promotion by name.",
		OutputSchema: mustOutputSchema[promotionResult](),
		Annotations:  destructive(),
	}, s.handleAbortPromotion)
}

// --- list_promotions ---

type listPromotionsArgs struct {
	Project string  `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"`
	Stage   *string `json:"stage,omitempty" jsonschema:"Filter to promotions targeting this stage"`
	Phase   string  `json:"phase,omitempty" jsonschema:"Filter by phase: Running, Succeeded, Failed, Errored, Pending, Aborted"`
}

// promotionJSON is the intake struct for summary projection.
type promotionJSON struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Stage   string `json:"stage"`
		Freight string `json:"freight"`
	} `json:"spec"`
	Status struct {
		Phase      string `json:"phase"`
		Message    string `json:"message"`
		StartedAt  string `json:"startedAt"`
		FinishedAt string `json:"finishedAt"`
	} `json:"status"`
}

type promotionSummary struct {
	Name       string `json:"name"`
	Stage      string `json:"stage,omitempty"`
	Freight    string `json:"freight,omitempty"`
	Phase      string `json:"phase,omitempty"`
	Message    string `json:"message,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

type promotionCondition struct {
	Type    string `json:"type,omitempty"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type promotionResult struct {
	Name       string               `json:"name,omitempty"`
	Project    string               `json:"namespace,omitempty"`
	Stage      string               `json:"stage,omitempty"`
	Freight    string               `json:"freight,omitempty"`
	Phase      string               `json:"phase,omitempty"`
	Message    string               `json:"message,omitempty"`
	Conditions []*promotionCondition `json:"conditions,omitempty"`
}

func promotionToSummary(p promotionJSON) promotionSummary {
	return promotionSummary{
		Name:       p.Metadata.Name,
		Stage:      p.Spec.Stage,
		Freight:    p.Spec.Freight,
		Phase:      p.Status.Phase,
		Message:    p.Status.Message,
		StartedAt:  p.Status.StartedAt,
		FinishedAt: p.Status.FinishedAt,
	}
}

func (s *Server) handleListPromotions(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listPromotionsArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	params := core.NewListPromotionsParams().WithProject(project)
	if args.Stage != nil {
		params = params.WithStage(args.Stage)
	}
	res, err := apiClient.Core.ListPromotions(params, nil)
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
	items := list.Items
	if args.Phase != "" {
		items = filterRawsByPhase(items, args.Phase)
	}
	summaries := projectItems(items, promotionToSummary)
	return jsonAnyResult(map[string]any{"items": summaries})
}

// filterRawsByPhase keeps only promotion JSON items whose status.phase matches
// (case-insensitive).
func filterRawsByPhase(raws []json.RawMessage, phase string) []json.RawMessage {
	var out []json.RawMessage
	for _, raw := range raws {
		var p promotionJSON
		if err := json.Unmarshal(raw, &p); err != nil {
			continue
		}
		if strings.EqualFold(p.Status.Phase, phase) {
			out = append(out, raw)
		}
	}
	return out
}

// --- get_promotion ---

type getPromotionArgs struct {
	Project   string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"`
	Promotion string `json:"promotion" jsonschema:"The name of the promotion"`
}

func (s *Server) handleGetPromotion(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getPromotionArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetPromotion(
		core.NewGetPromotionParams().WithProject(project).WithPromotion(args.Promotion),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(res.Payload))
}

// --- promote_to_stage ---

type promoteToStageArgs struct {
	Project      string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"`
	Stage        string `json:"stage" jsonschema:"The name of the stage to promote the freight to"`
	Freight      string `json:"freight,omitempty" jsonschema:"The name of the piece of freight to promote"`
	FreightAlias string `json:"freight_alias,omitempty" jsonschema:"The alias of the piece of freight to promote"`
}

func (s *Server) handlePromoteToStage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args promoteToStageArgs,
) (*mcp.CallToolResult, any, error) {
	if args.Freight == "" && args.FreightAlias == "" {
		return errResult(fmt.Errorf("either freight or freight_alias is required"))
	}
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.PromoteToStage(
		core.NewPromoteToStageParams().
			WithProject(project).
			WithStage(args.Stage).
			WithBody(&models.PromoteToStageRequest{
				Freight:      args.Freight,
				FreightAlias: args.FreightAlias,
			}),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(res.Payload))
}

// --- promote_downstream ---

type promoteDownstreamArgs struct {
	Project      string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"`
	Stage        string `json:"stage" jsonschema:"The upstream stage whose immediately downstream stages will receive the freight"`
	Freight      string `json:"freight,omitempty" jsonschema:"The name of the piece of freight to promote"`
	FreightAlias string `json:"freight_alias,omitempty" jsonschema:"The alias of the piece of freight to promote"`
}

func (s *Server) handlePromoteDownstream(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args promoteDownstreamArgs,
) (*mcp.CallToolResult, any, error) {
	if args.Freight == "" && args.FreightAlias == "" {
		return errResult(fmt.Errorf("either freight or freight_alias is required"))
	}
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.PromoteDownstream(
		core.NewPromoteDownstreamParams().
			WithProject(project).
			WithStage(args.Stage).
			WithBody(&models.PromoteDownstreamRequest{
				Freight:      args.Freight,
				FreightAlias: args.FreightAlias,
			}),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(sanitizeResource(res.Payload))
}

// --- abort_promotion ---

type abortPromotionArgs struct {
	Project   string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"`
	Promotion string `json:"promotion" jsonschema:"The name of the promotion to abort"`
}

func (s *Server) handleAbortPromotion(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args abortPromotionArgs,
) (*mcp.CallToolResult, any, error) {
	project, err := s.resolveProject(args.Project)
	if err != nil {
		return errResult(err)
	}
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	_, err = apiClient.Core.AbortPromotion(
		core.NewAbortPromotionParams().WithProject(project).WithPromotion(args.Promotion),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return okResult("Promotion aborted successfully.")
}
