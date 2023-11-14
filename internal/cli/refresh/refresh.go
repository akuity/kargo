package refresh

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh a stage or warehouse",
	}
	cmd.AddCommand(newRefreshWarehouseCommand(opt))
	cmd.AddCommand(newRefreshStageCommand(opt))
	return cmd
}

type Flags struct {
	Wait bool
}

const (
	refreshResourceTypeWarehouse = "warehouse"
	refreshResourceTypeStage     = "stage"
)

func addRefreshFlags(cmd *cobra.Command, flag *Flags) {
	cmd.Flags().BoolVar(&flag.Wait, "wait", true, "Wait until refresh completes")
}

func refreshObject(
	opt *option.Option,
	flag *Flags,
	resourceType string,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
		if err != nil {
			return err
		}

		project := opt.Project.OrElse("")
		if project == "" {
			return errors.New("project is required")
		}
		name := strings.TrimSpace(args[0])
		if name == "" {
			return errors.New("name is required")
		}

		switch resourceType {
		case refreshResourceTypeWarehouse:
			_, err = kargoSvcCli.RefreshWarehouse(ctx, connect.NewRequest(&v1alpha1.RefreshWarehouseRequest{
				Project: project,
				Name:    name,
			}))

		case refreshResourceTypeStage:
			_, err = kargoSvcCli.RefreshStage(ctx, connect.NewRequest(&v1alpha1.RefreshStageRequest{
				Project: project,
				Name:    name,
			}))
		}
		if err != nil {
			return errors.Wrapf(err, "refresh %s", resourceType)
		}

		if flag.Wait {
			switch resourceType {
			case refreshResourceTypeWarehouse:
				err = waitForWarehouse(ctx, kargoSvcCli, project, name)
			case refreshResourceTypeStage:
				err = waitForStage(ctx, kargoSvcCli, project, name)
			}
			if err != nil {
				return errors.Wrapf(err, "wait %s", resourceType)
			}
		}
		fmt.Printf("%s '%s/%s' refreshed\n", resourceType, project, name)
		return nil
	}
}
