package refresh

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/server"
)

type refreshOptions struct {
	Config        config.CLIConfig
	ClientOptions client.Options

	Project      string
	ResourceType server.RefreshResourceType
	Name         string
	Wait         bool
}

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh TYPE NAME [--wait]",
		Short: "Refresh a Kargo resource",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Refresh a warehouse
kargo refresh warehouse --project=my-project my-warehouse

# Refresh a stage
kargo refresh stage --project=my-project my-stage

# Refresh the cluster configuration
kargo refresh clusterconfig

# Refresh the project configuration
kargo refresh projectconfig --project=my-project
`),
	}

	// Register subcommands.
	cmd.AddCommand(newRefreshClusterConfigCommand(cfg))
	cmd.AddCommand(newRefreshProjectConfigCommand(cfg))
	cmd.AddCommand(newRefreshStageCommand(cfg))
	cmd.AddCommand(newRefreshWarehouseCommand(cfg))

	return cmd
}

// addFlags adds the flags for the refresh options to the provided command.
func (o *refreshOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())

	if o.ResourceType.IsNamespaced() {
		option.Project(cmd.Flags(), &o.Project, o.Config.Project,
			"The Project the resource belongs to. If not set, the default project will be used.")
	}
	option.Wait(cmd.Flags(), &o.Wait, false, "Wait for the refresh to complete.")
}

// complete sets the resource type for the refresh options, and further parses
// the command arguments to set the resource name.
func (o *refreshOptions) complete(args []string) {
	if o.ResourceType.IsNamespaced() && !o.ResourceType.NameEqualsProject() {
		o.Name = strings.TrimSpace(args[0])
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *refreshOptions) validate() error {
	var errs []error

	if o.ResourceType == "" {
		errs = append(errs, errors.New("resource type is required"))
	}

	if o.ResourceType.IsNamespaced() && o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	if !o.ResourceType.NameEqualsProject() && o.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	return errors.Join(errs...)
}

// run performs the refresh operation based on the provided options.
func (o *refreshOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	req := connect.NewRequest(&v1alpha1.RefreshResourceRequest{
		Project:      o.Project,
		Name:         o.Name,
		ResourceType: o.ResourceType.String(),
	})
	if _, err = kargoSvcCli.RefreshResource(ctx, req); err != nil {
		return fmt.Errorf("refresh %s: %w", o.ResourceType, err)
	}

	if o.Wait {
		switch o.ResourceType {
		case server.RefreshResourceTypeClusterConfig:
			err = waitForClusterConfig(ctx, kargoSvcCli)
		case server.RefreshResourceTypeProjectConfig:
			err = waitForProjectConfig(ctx, kargoSvcCli, o.Project)
		case server.RefreshResourceTypeStage:
			err = waitForStage(ctx, kargoSvcCli, o.Project, o.Name)
		case server.RefreshResourceTypeWarehouse:
			err = waitForWarehouse(ctx, kargoSvcCli, o.Project, o.Name)
		}
		if err != nil {
			return fmt.Errorf("wait %s: %w", o.ResourceType, err)
		}
	}
	fmt.Printf("%s '%s' refreshed\n", o.ResourceType, formatName(o.Project, o.Name))
	return nil
}

func formatName(project, name string) string {
	if project == "" {
		return name
	}
	if name == "" {
		return project
	}
	return fmt.Sprintf("%s/%s", project, name)
}
