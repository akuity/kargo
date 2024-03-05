package refresh

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

const (
	refreshResourceTypeWarehouse = "warehouse"
	refreshResourceTypeStage     = "stage"
)

type refreshOptions struct {
	*option.Option
	Config config.CLIConfig

	Project      string
	ResourceType string
	Name         string
	Wait         bool
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh TYPE NAME [--wait]",
		Short: "Refresh a stage or warehouse",
		Args:  option.NoArgs,
		Example: `
# Refresh a warehouse
kargo refresh warehouse --project=my-project my-warehouse

# Refresh a stage
kargo refresh stage --project=my-project my-stage
`,
	}

	// Register subcommands.
	cmd.AddCommand(newRefreshWarehouseCommand(cfg, opt))
	cmd.AddCommand(newRefreshStageCommand(cfg, opt))

	return cmd
}

// addFlags adds the flags for the refresh options to the provided command.
func (o *refreshOptions) addFlags(cmd *cobra.Command) {
	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

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

	return goerrors.Join(errs...)
}

// run performs the refresh operation based on the provided options.
func (o *refreshOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return err
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
		return errors.Wrapf(err, "refresh %s", o.ResourceType)
	}

	if o.Wait {
		switch o.ResourceType {
		case refreshResourceTypeWarehouse:
			err = waitForWarehouse(ctx, kargoSvcCli, o.Project, o.Name)
		case refreshResourceTypeStage:
			err = waitForStage(ctx, kargoSvcCli, o.Project, o.Name)
		}
		if err != nil {
			return errors.Wrapf(err, "wait %s", o.ResourceType)
		}
	}
	fmt.Printf("%s '%s/%s' refreshed\n", o.ResourceType, o.Project, o.Name)
	return nil
}
