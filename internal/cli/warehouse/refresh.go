package warehouse

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

type RefreshFlags struct {
	Wait bool
}

func newRefreshCommand(opt *option.Option) *cobra.Command {
	//var flag RefreshFlags
	cmd := &cobra.Command{
		Use:     "refresh (PROJECT) (WAREHOUSE)",
		Args:    option.ExactArgs(2),
		Example: "kargo warehouse refresh (PROJECT) (WAREHOUSE)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return err
			}

			project := strings.TrimSpace(args[0])
			if project == "" {
				return errors.New("project is required")
			}
			name := strings.TrimSpace(args[1])
			if name == "" {
				return errors.New("name is required")
			}

			_, err = kargoSvcCli.RefreshWarehouse(ctx, connect.NewRequest(&v1alpha1.RefreshWarehouseRequest{
				Project: project,
				Name:    name,
			}))
			if err != nil {
				return errors.Wrap(err, "refresh warehouse")
			}
			// if flag.Wait {
			// 	// TODO: wait until annotation clears
			// }
			fmt.Printf("Warehouse '%s/%s' refreshed\n", project, name)
			return nil
		},
	}
	//cmd.Flags().BoolVarP(&flag.Wait, "wait", "w", true, "Wait until refresh completes")
	return cmd
}
