package kargomcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/core"
)

func (s *Server) registerFreightTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "list_freight",
		Description: "List freight in a Kargo project. Optionally filter by stage " +
			"(freight currently in that stage) or by origin warehouse.",
		OutputSchema: mustOutputSchema[freightListResult](),
		Annotations:  readOnly(),
	}, s.handleListFreight)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_freight",
		Description:  "Get a single piece of freight by name or alias within a Kargo project.",
		OutputSchema: mustOutputSchema[freightResult](),
		Annotations:  readOnly(),
	}, s.handleGetFreight)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "approve_freight",
		Description: "Manually approve a piece of freight for promotion to a specific stage, " +
			"bypassing the normal verification process. Requires promote permission on the stage.",
		OutputSchema: mustOutputSchema[freightResult](),
		Annotations:  destructive(),
	}, s.handleApproveFreight)
}

// --- list_freight ---

type listFreightArgs struct {
	Project  string  `json:"project" jsonschema:"The name of the Kargo project"`
	Stage    *string `json:"stage,omitempty" jsonschema:"Filter to freight currently in this stage"`
	Origins  []string `json:"origins,omitempty" jsonschema:"Filter by origin warehouse names"`
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

type freightListResult struct {
	Items []*freightResult `json:"items,omitempty"`
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
	return jsonAnyResult(res.Payload)
}

// --- get_freight ---

type getFreightArgs struct {
	Project         string `json:"project" jsonschema:"The name of the Kargo project"`
	FreightNameOrAlias string `json:"freight" jsonschema:"The name or alias of the piece of freight"`
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
	Project         string `json:"project" jsonschema:"The name of the Kargo project"`
	FreightNameOrAlias string `json:"freight" jsonschema:"The name or alias of the piece of freight to approve"`
	Stage           string `json:"stage" jsonschema:"The name of the stage to approve the freight for"`
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
