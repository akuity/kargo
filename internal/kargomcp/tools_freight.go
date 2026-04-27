package kargomcp

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerFreightTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_freight",
		Description: "List freight in a Kargo project, newest first. Returns a compact summary per piece. " +
			"Optionally filter by stage (freight currently in that stage) or by origin warehouse.",
		OutputSchema: mustOutputSchema[struct {
			Items []freightSummary `json:"items"`
		}](),
		Annotations:  readOnly(),
	}, s.handleListFreight)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_freight",
		Description:  "Get full details for a single piece of freight by name or alias.",
		OutputSchema: mustOutputSchema[freightResult](),
		Annotations:  readOnly(),
	}, s.handleGetFreight)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "approve_freight",
		Description: "Manually approve a piece of freight for promotion to a specific stage, " +
			"bypassing the normal verification process. Requires promote permission on the stage.",
		Annotations: destructive(),
	}, s.handleApproveFreight)
}

// --- list_freight ---

type listFreightArgs struct {
	Project string   `json:"project" jsonschema:"The name of the Kargo project"`
	Stage   *string  `json:"stage,omitempty" jsonschema:"Filter to freight currently in this stage"`
	Origins []string `json:"origins,omitempty" jsonschema:"Filter by origin warehouse names"`
	Limit   int      `json:"limit,omitempty" jsonschema:"Max number to return, newest first (default 20)"`
}

// freightJSON is the intake struct for summary projection.
type freightJSON struct {
	Alias    string `json:"alias"`
	Metadata struct {
		Name              string `json:"name"`
		CreationTimestamp string `json:"creationTimestamp"`
	} `json:"metadata"`
	Origin struct {
		Name string `json:"name"`
	} `json:"origin"`
	Commits []struct {
		RepoURL string `json:"repoURL"`
		ID      string `json:"id"`
		Tag     string `json:"tag"`
		Message string `json:"message"`
	} `json:"commits"`
	Images []struct {
		RepoURL string `json:"repoURL"`
		Tag     string `json:"tag"`
	} `json:"images"`
	Charts []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"charts"`
	Status struct {
		CurrentlyIn map[string]json.RawMessage `json:"currentlyIn"`
	} `json:"status"`
}

type freightSummaryImage struct {
	RepoURL string `json:"repoURL,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type freightSummaryCommit struct {
	RepoURL string `json:"repoURL,omitempty"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

type freightSummaryChart struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type freightSummary struct {
	Name      string                `json:"name"`
	Alias     string                `json:"alias,omitempty"`
	CreatedAt string                `json:"createdAt,omitempty"`
	Warehouse string                `json:"warehouse,omitempty"`
	Stages    []string              `json:"stages,omitempty"`
	Images    []freightSummaryImage  `json:"images,omitempty"`
	Commits   []freightSummaryCommit `json:"commits,omitempty"`
	Charts    []freightSummaryChart  `json:"charts,omitempty"`
}

func freightToSummary(f freightJSON) freightSummary {
	s := freightSummary{
		Name:      f.Metadata.Name,
		Alias:     f.Alias,
		CreatedAt: f.Metadata.CreationTimestamp,
		Warehouse: f.Origin.Name,
	}
	for stageName := range f.Status.CurrentlyIn {
		s.Stages = append(s.Stages, stageName)
	}
	for _, img := range f.Images {
		s.Images = append(s.Images, freightSummaryImage{RepoURL: img.RepoURL, Tag: img.Tag})
	}
	for _, c := range f.Commits {
		s.Commits = append(s.Commits, freightSummaryCommit{RepoURL: c.RepoURL, ID: c.ID, Message: c.Message})
	}
	for _, ch := range f.Charts {
		s.Charts = append(s.Charts, freightSummaryChart{Name: ch.Name, Version: ch.Version})
	}
	return s
}

func (s *Server) handleListFreight(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args listFreightArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	params := core.NewQueryFreightsRestParams().WithProject(args.Project)
	if args.Stage != nil {
		params = params.WithStage(args.Stage)
	}
	if len(args.Origins) > 0 {
		params = params.WithOrigins(args.Origins)
	}
	res, err := apiClient.Core.QueryFreightsRest(params, nil)
	if err != nil {
		return errResult(err)
	}
	raws := flattenFreightGroups(res.Payload)
	summaries := projectItems(raws, args.Limit, freightToSummary)
	return jsonAnyResult(map[string]any{"items": summaries})
}

// flattenFreightGroups collapses the QueryFreightsRest grouped response
// ({"groups":{"":{"items":[...]}}}) into a flat []json.RawMessage.
func flattenFreightGroups(payload any) []json.RawMessage {
	data, _ := json.Marshal(payload)
	var grouped struct {
		Groups map[string]struct {
			Items []json.RawMessage `json:"items"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(data, &grouped); err != nil {
		return nil
	}
	var items []json.RawMessage
	for _, g := range grouped.Groups {
		items = append(items, g.Items...)
	}
	return items
}

// --- get_freight ---

type getFreightArgs struct {
	Project            string `json:"project" jsonschema:"The name of the Kargo project"`
	FreightNameOrAlias string `json:"freight" jsonschema:"The name or alias of the piece of freight"`
}

type freightCommit struct {
	RepoURL string `json:"repoURL,omitempty"`
	ID      string `json:"id,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Message string `json:"message,omitempty"`
	Author  string `json:"author,omitempty"`
}

type freightImage struct {
	RepoURL string `json:"repoURL,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

type freightChart struct {
	RepoURL string `json:"repoURL,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type freightOrigin struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

type freightResult struct {
	Name    string          `json:"name,omitempty"`
	Alias   string          `json:"alias,omitempty"`
	Project string          `json:"namespace,omitempty"`
	Origin  *freightOrigin  `json:"origin,omitempty"`
	Commits []*freightCommit `json:"commits,omitempty"`
	Images  []*freightImage  `json:"images,omitempty"`
	Charts  []*freightChart  `json:"charts,omitempty"`
}

func (s *Server) handleGetFreight(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args getFreightArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.Core.GetFreight(
		core.NewGetFreightParams().
			WithProject(args.Project).
			WithFreightNameOrAlias(args.FreightNameOrAlias),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(res.Payload)
}

// --- approve_freight ---

type approveFreightArgs struct {
	Project            string `json:"project" jsonschema:"The name of the Kargo project"`
	FreightNameOrAlias string `json:"freight" jsonschema:"The name or alias of the piece of freight to approve"`
	Stage              string `json:"stage" jsonschema:"The name of the stage to approve the freight for"`
}

func (s *Server) handleApproveFreight(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args approveFreightArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	_, err = apiClient.Core.ApproveFreight(
		core.NewApproveFreightParams().
			WithProject(args.Project).
			WithFreightNameOrAlias(args.FreightNameOrAlias).
			WithStage(args.Stage),
		nil,
	)
	if err != nil {
		return errResult(err)
	}
	return okResult("Freight approved successfully.")
}
