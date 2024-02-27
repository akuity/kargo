package refresh

import (
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

type refreshOptions struct {
	*option.Option

	Wait bool
}

// addFlags adds the flags for the refresh options to the provided command.
func (o *refreshOptions) addFlags(cmd *cobra.Command) {
	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

	option.Project(cmd.PersistentFlags(), &o.Project, o.Project,
		"The Project the resource belongs to. If not set, the default project will be used.")
	option.Wait(cmd.PersistentFlags(), &o.Wait, false, "Wait for the refresh to complete.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *refreshOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &refreshOptions{Option: opt}

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh a stage or warehouse",
	}

	cmd.AddCommand(newRefreshWarehouseCommand(cfg, cmdOpts))
	cmd.AddCommand(newRefreshStageCommand(cfg, cmdOpts))

	// Register the option flags on the (root) command.
	cmdOpts.addFlags(cmd)

	return cmd
}

const (
	refreshResourceTypeWarehouse = "warehouse"
	refreshResourceTypeStage     = "stage"
)

func refreshObject(
	cfg config.CLIConfig,
	opt *refreshOptions,
	resourceType string,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		name := strings.TrimSpace(args[0])
		if name == "" {
			return errors.New("name is required")
		}

		if err := opt.validate(); err != nil {
			return err
		}

		kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt.Option)
		if err != nil {
			return err
		}

		switch resourceType {
		case refreshResourceTypeWarehouse:
			_, err = kargoSvcCli.RefreshWarehouse(ctx, connect.NewRequest(&v1alpha1.RefreshWarehouseRequest{
				Project: opt.Project,
				Name:    name,
			}))

		case refreshResourceTypeStage:
			_, err = kargoSvcCli.RefreshStage(ctx, connect.NewRequest(&v1alpha1.RefreshStageRequest{
				Project: opt.Project,
				Name:    name,
			}))
		}
		if err != nil {
			return errors.Wrapf(err, "refresh %s", resourceType)
		}

		if opt.Wait {
			switch resourceType {
			case refreshResourceTypeWarehouse:
				err = waitForWarehouse(ctx, kargoSvcCli, opt.Project, name)
			case refreshResourceTypeStage:
				err = waitForStage(ctx, kargoSvcCli, opt.Project, name)
			}
			if err != nil {
				return errors.Wrapf(err, "wait %s", resourceType)
			}
		}
		fmt.Printf("%s '%s/%s' refreshed\n", resourceType, opt.Project, name)
		return nil
	}
}
