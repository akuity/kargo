package completion

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

// timeout bounds how long a completion request may block the shell.
var timeout = 5 * time.Second

// ProjectNames returns a completion function that suggests the names of
// Projects from the Kargo API server. It suggests nothing if the user is not
// logged in or the request fails.
func ProjectNames(cfg config.CLIConfig) cobra.CompletionFunc {
	return listNames(
		cfg,
		func(
			ctx context.Context,
			c *kargogen.APIClient,
			_ string,
		) ([]string, *http.Response, error) {
			res, httpRes, err := c.CoreAPI.ListProjects(ctx).Execute()
			var names []string
			for _, project := range res.GetItems() {
				names = append(names, project.Metadata.GetName())
			}
			return names, httpRes, err
		},
	)
}

// StageNames returns a completion function that suggests the names of Stages
// in the Project from the project flag, or in the default Project.
func StageNames(cfg config.CLIConfig) cobra.CompletionFunc {
	return listNames(
		cfg,
		func(
			ctx context.Context,
			c *kargogen.APIClient,
			project string,
		) ([]string, *http.Response, error) {
			res, httpRes, err := c.CoreAPI.ListStages(ctx, project).Execute()
			var names []string
			for _, stage := range res.GetItems() {
				names = append(names, stage.Metadata.GetName())
			}
			return names, httpRes, err
		},
	)
}

// WarehouseNames returns a completion function that suggests the names of
// Warehouses in the Project from the project flag, or in the default Project.
func WarehouseNames(cfg config.CLIConfig) cobra.CompletionFunc {
	return listNames(
		cfg,
		func(
			ctx context.Context,
			c *kargogen.APIClient,
			project string,
		) ([]string, *http.Response, error) {
			res, httpRes, err := c.CoreAPI.ListWarehouses(ctx, project).Execute()
			var names []string
			for _, warehouse := range res.GetItems() {
				names = append(names, warehouse.Metadata.GetName())
			}
			return names, httpRes, err
		},
	)
}

// listNames returns a completion function that suggests the names returned by
// list. It suggests nothing if the user is not logged in or the request fails.
func listNames(
	cfg config.CLIConfig,
	list func(context.Context, *kargogen.APIClient, string) ([]string, *http.Response, error),
) cobra.CompletionFunc {
	return func(cmd *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		insecure, _ := cmd.Flags().GetBool(option.InsecureTLSFlag)
		apiClient, err := client.GetClientFromConfig(ctx, cfg, client.Options{InsecureTLS: insecure})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// The project flag defaults to the default Project.
		project, _ := cmd.Flags().GetString(option.ProjectFlag)
		names, httpRes, err := list(ctx, apiClient, project)
		if httpRes != nil {
			_ = httpRes.Body.Close()
		}
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
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
