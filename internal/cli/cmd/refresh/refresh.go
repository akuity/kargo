package refresh

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

const (
	refreshResourceTypeWarehouse = "warehouse"
	refreshResourceTypeStage     = "stage"
)

type refreshOptions struct {
	Config        config.CLIConfig
	ClientOptions client.Options

	Project      string
	ResourceType string
	Name         string
	Wait         bool
}

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh TYPE NAME [--wait]",
		Short: "Refresh a stage or warehouse",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Refresh a warehouse
kargo refresh warehouse --project=my-project my-warehouse

# Refresh a stage
kargo refresh stage --project=my-project my-stage
`),
	}

	// Register subcommands.
	cmd.AddCommand(newRefreshWarehouseCommand(cfg))
	cmd.AddCommand(newRefreshStageCommand(cfg))

	return cmd
}

// addFlags adds the flags for the refresh options to the provided command.
func (o *refreshOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project the resource belongs to. If not set, the default project will be used.")
	option.Wait(cmd.Flags(), &o.Wait, false, "Wait for the refresh to complete.")
}

// complete sets the resource type for the refresh options, and further parses
// the command arguments to set the resource name.
func (o *refreshOptions) complete(resourceType string, args []string) {
	o.ResourceType = resourceType
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *refreshOptions) validate() error {
	var errs []error

	if o.ResourceType == "" {
		errs = append(errs, errors.New("resource type is required"))
	}

	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	if o.Name == "" {
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

	switch o.ResourceType {
	case refreshResourceTypeWarehouse:
		_, err = kargoSvcCli.RefreshWarehouse(ctx, connect.NewRequest(&v1alpha1.RefreshWarehouseRequest{
			Project: o.Project,
			Name:    o.Name,
		}))

	case refreshResourceTypeStage:
		_, err = kargoSvcCli.RefreshStage(ctx, connect.NewRequest(&v1alpha1.RefreshStageRequest{
			Project: o.Project,
			Name:    o.Name,
		}))
	}
	if err != nil {
		return fmt.Errorf("refresh %s: %w", o.ResourceType, err)
	}

	if o.Wait {
		switch o.ResourceType {
		case refreshResourceTypeWarehouse:
			err = waitForWarehouse(ctx, kargoSvcCli, o.Project, o.Name)
		case refreshResourceTypeStage:
			err = waitForStage(ctx, kargoSvcCli, o.Project, o.Name)
		}
		if err != nil {
			return fmt.Errorf("wait %s: %w", o.ResourceType, err)
		}
	}
	fmt.Printf("%s '%s/%s' refreshed\n", o.ResourceType, o.Project, o.Name)
	return nil
}
