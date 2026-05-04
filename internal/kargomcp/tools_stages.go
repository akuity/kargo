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
		Name:         "get_stage",
		Description:  "Get full details for a single stage.",
		OutputSchema: mustOutputSchema[stageResult](),
		Annotations:  readOnly(),
	}, s.handleGetStage)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "get_stage_freight_history",
		Description: "Get the freight history for a stage — the sequence of freight " +
			"collections that have been promoted through it, with verification results. " +
			"Use this when you need to audit what freight has passed through a stage " +
			"or investigate verification failures.",
		OutputSchema: mustOutputSchema[struct {
			Items []freightHistoryEntry `json:"items"`
		}](),
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

// stageJSON is the intake struct for summary projection.
type stageJSON struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Status struct {
		Health struct {
			Status string   `json:"status"`
			Issues []string `json:"issues"`
		} `json:"health"`
		FreightSummary   string `json:"freightSummary"`
		CurrentPromotion struct {
			Name string `json:"name"`
		} `json:"currentPromotion"`
		LastPromotion struct {
			Name       string `json:"name"`
			FinishedAt string `json:"finishedAt"`
			Status     struct {
				Phase   string `json:"phase"`
				Message string `json:"message"`
			} `json:"status"`
		} `json:"lastPromotion"`
	} `json:"status"`
}

type stageSummary struct {
	Name             string              `json:"name"`
	Health           string              `json:"health,omitempty"`
	HealthIssues     []string            `json:"healthIssues,omitempty"`
	CurrentFreight   string              `json:"currentFreight,omitempty"`
	CurrentPromotion string              `json:"currentPromotion,omitempty"`
	LastPromotion    *lastPromotionBrief `json:"lastPromotion,omitempty"`
}

type lastPromotionBrief struct {
	Name       string `json:"name,omitempty"`
	Phase      string `json:"phase,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Message    string `json:"message,omitempty"`
}

func stageToSummary(st stageJSON) stageSummary {
	s := stageSummary{
		Name:             st.Metadata.Name,
		Health:           st.Status.Health.Status,
		HealthIssues:     st.Status.Health.Issues,
		CurrentFreight:   st.Status.FreightSummary,
		CurrentPromotion: st.Status.CurrentPromotion.Name,
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
	data, _ := json.Marshal(res.Payload)
	var list struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return errResult(err)
	}

	// Filter by health before projecting.
	want := strings.ToLower(args.Health)
	filtered := list.Items
	if want != "" {
		filtered = filtered[:0]
		for _, raw := range list.Items {
			var st stageJSON
			if err := json.Unmarshal(raw, &st); err != nil {
				continue
			}
			if strings.EqualFold(st.Status.Health.Status, want) {
				filtered = append(filtered, raw)
			}
		}
	}

	summaries := make([]stageSummary, 0, len(filtered))
	for _, raw := range filtered {
		var st stageJSON
		if err := json.Unmarshal(raw, &st); err != nil {
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
	sanitized, ok := sanitizeResource(res.Payload).(map[string]any)
	if !ok {
		return jsonAnyResult(sanitizeResource(res.Payload))
	}
	if status, ok := sanitized["status"].(map[string]any); ok {
		delete(status, "freightHistory")
	}
	return jsonAnyResult(sanitized)
}

// --- get_stage_freight_history ---

type getStageFreightHistoryArgs struct {
	Project string `json:"project,omitempty" jsonschema:"The Kargo project name. Omit to use the default set by 'kargo config set-project'"` //nolint:lll
	Stage   string `json:"stage" jsonschema:"The name of the stage"`
}

type freightHistoryVerification struct {
	ID         string `json:"id,omitempty"`
	Phase      string `json:"phase,omitempty"`
	StartTime  string `json:"startTime,omitempty"`
	FinishTime string `json:"finishTime,omitempty"`
}

type freightHistoryEntry struct {
	ID                  string                       `json:"id,omitempty"`
	FreightName         string                       `json:"freightName,omitempty"`
	VerificationHistory []freightHistoryVerification `json:"verificationHistory,omitempty"`
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
	data, _ := json.Marshal(res.Payload)
	var stage struct {
		Status struct {
			FreightHistory []struct {
				ID    string `json:"id"`
				Items map[string]struct {
					Name string `json:"name"`
				} `json:"items"`
				VerificationHistory []freightHistoryVerification `json:"verificationHistory"`
			} `json:"freightHistory"`
		} `json:"status"`
	}
	if err := json.Unmarshal(data, &stage); err != nil {
		return errResult(err)
	}
	entries := make([]freightHistoryEntry, 0, len(stage.Status.FreightHistory))
	for _, h := range stage.Status.FreightHistory {
		e := freightHistoryEntry{
			ID:                  h.ID,
			VerificationHistory: h.VerificationHistory,
		}
		// Extract freight name from the first item (there is typically one per origin).
		for _, item := range h.Items {
			if item.Name != "" {
				e.FreightName = item.Name
				break
			}
		}
		entries = append(entries, e)
	}
	return jsonAnyResult(map[string]any{"items": entries})
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
