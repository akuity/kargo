package kargomcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/client/generated/system"
)

func (s *Server) registerServerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "get_version_info",
		Description:  "Get the Kargo server version information.",
		OutputSchema: mustOutputSchema[versionInfoResult](),
		Annotations:  readOnly(),
	}, s.handleGetVersionInfo)
}

type getVersionInfoArgs struct{}

type versionInfoResult struct {
	Version      string `json:"version,omitempty"`
	GitCommit    string `json:"gitCommit,omitempty"`
	GitTreeDirty bool   `json:"gitTreeDirty,omitempty"`
	BuildDate    string `json:"buildDate,omitempty"`
	GoVersion    string `json:"goVersion,omitempty"`
	Platform     string `json:"platform,omitempty"`
}

func (s *Server) handleGetVersionInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ getVersionInfoArgs,
) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.apiClient(ctx)
	if err != nil {
		return errResult(err)
	}
	res, err := apiClient.System.GetVersionInfo(system.NewGetVersionInfoParams(), nil)
	if err != nil {
		return errResult(err)
	}
	return jsonAnyResult(res.Payload)
}
