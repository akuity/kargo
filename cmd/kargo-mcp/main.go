package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	generatedclient "github.com/akuity/kargo/pkg/client/generated"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/generated/system"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := loadConfig()

	s := server.NewMCPServer(
		"kargo-mcp",
		"0.1.0",
		server.WithInstructions(
			"Kargo MCP server. Use these tools to query and manage Kargo "+
				"continuous promotion pipelines. If you receive an auth error, "+
				"ask the user to run `kargo login`.",
		),
	)

	registerTools(s, cfg)

	logger.Info("kargo-mcp starting", "address", cfg.APIAddress)
	if err := server.ServeStdio(s); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// loadConfig merges environment variables and the CLI config file.
// Environment variables take priority.
func loadConfig() config.CLIConfig {
	env := config.NewEnvVarCLIConfig()

	// Load the file config and fill any gaps not covered by env vars.
	file, err := config.LoadCLIConfig()
	if err == nil {
		if env.APIAddress == "" {
			env.APIAddress = file.APIAddress
		}
		if env.BearerToken == "" {
			env.BearerToken = file.BearerToken
			env.RefreshToken = file.RefreshToken
			env.InsecureSkipTLSVerify = file.InsecureSkipTLSVerify
		}
	}
	return env
}

// getAPIClient constructs an authenticated API client, handling token refresh.
// Returns a user-friendly error if authentication is missing or expired.
func getAPIClient(ctx context.Context, cfg config.CLIConfig) (*generatedclient.KargoAPI, error) {
	apiClient, err := client.GetClientFromConfig(ctx, cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("%w\nRun `kargo login` to authenticate", err)
	}
	return apiClient, nil
}

// toolError returns a CallToolResult representing an error.
func toolError(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultErrorf(format, args...), nil
}

func registerTools(s *server.MCPServer, cfg config.CLIConfig) {
	s.AddTool(
		mcp.NewTool("get_version_info",
			mcp.WithDescription("Get the Kargo server version information."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			apiClient, err := getAPIClient(ctx, cfg)
			if err != nil {
				return toolError("%v", err)
			}
			res, err := apiClient.System.GetVersionInfo(
				system.NewGetVersionInfoParams(),
				nil,
			)
			if err != nil {
				return toolError("get version info: %v", err)
			}
			out, err := json.MarshalIndent(res.Payload, "", "  ")
			if err != nil {
				return toolError("marshal response: %v", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("List all Kargo projects the authenticated user has access to."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			apiClient, err := getAPIClient(ctx, cfg)
			if err != nil {
				return toolError("%v", err)
			}
			res, err := apiClient.Core.ListProjects(
				core.NewListProjectsParams(),
				nil,
			)
			if err != nil {
				return toolError("list projects: %v", err)
			}
			out, err := json.MarshalIndent(res.Payload, "", "  ")
			if err != nil {
				return toolError("marshal response: %v", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("list_stages",
			mcp.WithDescription("List all stages in a Kargo project."),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("The name of the Kargo project."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			project, err := req.RequireString("project")
			if err != nil {
				return toolError("project is required: %v", err)
			}
			apiClient, err := getAPIClient(ctx, cfg)
			if err != nil {
				return toolError("%v", err)
			}
			res, err := apiClient.Core.ListStages(
				core.NewListStagesParams().WithProject(project),
				nil,
			)
			if err != nil {
				return toolError("list stages: %v", err)
			}
			out, err := json.MarshalIndent(res.Payload, "", "  ")
			if err != nil {
				return toolError("marshal response: %v", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("promote_to_stage",
			mcp.WithDescription(
				"Promote a piece of freight to a specific stage in a Kargo project. "+
					"Provide either freight (name) or freight_alias, not both.",
			),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("The name of the Kargo project."),
			),
			mcp.WithString("stage",
				mcp.Required(),
				mcp.Description("The name of the stage to promote the freight to."),
			),
			mcp.WithString("freight",
				mcp.Description("The name of the piece of freight to promote."),
			),
			mcp.WithString("freight_alias",
				mcp.Description("The alias of the piece of freight to promote."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			project, err := req.RequireString("project")
			if err != nil {
				return toolError("project is required: %v", err)
			}
			stage, err := req.RequireString("stage")
			if err != nil {
				return toolError("stage is required: %v", err)
			}
			freight := req.GetString("freight", "")
			freightAlias := req.GetString("freight_alias", "")
			if freight == "" && freightAlias == "" {
				return toolError("either freight or freight_alias is required")
			}

			apiClient, err := getAPIClient(ctx, cfg)
			if err != nil {
				return toolError("%v", err)
			}
			res, err := apiClient.Core.PromoteToStage(
				core.NewPromoteToStageParams().
					WithProject(project).
					WithStage(stage).
					WithBody(&models.PromoteToStageRequest{
						Freight:      freight,
						FreightAlias: freightAlias,
					}),
				nil,
			)
			if err != nil {
				return toolError("promote to stage: %v", err)
			}
			out, err := json.MarshalIndent(res.Payload, "", "  ")
			if err != nil {
				return toolError("marshal response: %v", err)
			}
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
