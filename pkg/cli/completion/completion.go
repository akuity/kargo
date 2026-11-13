package completion

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
)

// timeout bounds how long a completion request may block the shell.
var timeout = 5 * time.Second

// ProjectNames returns a completion function that suggests the names of
// Projects from the Kargo API server. It suggests nothing if the user is not
// logged in or the request fails.
func ProjectNames(cfg config.CLIConfig) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		insecure, _ := cmd.Flags().GetBool(option.InsecureTLSFlag)
		apiClient, err := client.GetClientFromConfig(ctx, cfg, client.Options{InsecureTLS: insecure})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		res, httpRes, err := apiClient.CoreAPI.ListProjects(ctx).Execute()
		if httpRes != nil {
			_ = httpRes.Body.Close()
		}
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names := make([]cobra.Completion, 0, len(res.Items))
		for _, project := range res.Items {
			names = append(names, project.Metadata.GetName())
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// RegisterProjectFlag registers ProjectNames as the completion function for
// the project flag on the provided command and all of its subcommands.
func RegisterProjectFlag(cmd *cobra.Command, cfg config.CLIConfig) {
	if cmd.Flags().Lookup(option.ProjectFlag) != nil {
		// Cannot fail: the flag exists and each command is visited once.
		_ = cmd.RegisterFlagCompletionFunc(option.ProjectFlag, ProjectNames(cfg))
	}
	for _, sub := range cmd.Commands() {
		RegisterProjectFlag(sub, cfg)
	}
}
