package get

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type getFreightOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
	Aliases []string
	Origins []string
}

func newGetFreightCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,

) *cobra.Command {
	cmdOpts := &getFreightOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "freight [--project=project] [--name=name | --alias=alias] [--no-headers]",
		Short: "Display one or many pieces of freight",
		Args:  option.NoArgs,
		Example: templates.Example(`
# List all freight in my-project
kargo get freight --project=my-project

# List all freight in my-project for a specific warehouse
kargo get freight --project=my-project --origin=warehouse-1

# List all freight in my-project in JSON output format
kargo get freight --project=my-project -o json

# Get a single piece of freight by name
kargo get freight --project=my-project --name=abc1234

# Get a single piece of freight by alias
kargo get freight --project=my-project --alias=wonky-wombat

# List all freight in the default project
kargo config set-project my-project
kargo get freight

# Get a single piece of freight by name in the default project
kargo config set-project my-project
kargo get freight --name=abc1234

# Get a single piece of freight by alias in the default project
kargo config set-project my-project
kargo get freight --alias=wonky-wombat
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the get freight options to the provided command.
func (o *getFreightOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to get freight. If not set, the default project will be used.",
	)
	option.Names(cmd.Flags(), &o.Names, "The name of a piece of freight to get.")
	option.Aliases(cmd.Flags(), &o.Aliases, "The alias of a piece of freight to get.")
	option.Origins(cmd.Flags(), &o.Origins, "The origin of the freight to get.")

	// Origin and name/alias are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.NameFlag, option.OriginFlag)
	cmd.MarkFlagsMutuallyExclusive(option.AliasFlag, option.OriginFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getFreightOptions) validate() error {
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		return fmt.Errorf("%s is required", option.ProjectFlag)
	}
	return nil
}

// run gets the freight from the server and prints it to the console.
func (o *getFreightOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 && len(o.Aliases) == 0 {
		var resp *connect.Response[v1alpha1.QueryFreightResponse]
		if resp, err = kargoSvcCli.QueryFreight(
			ctx,
			connect.NewRequest(
				&v1alpha1.QueryFreightRequest{
					Project: o.Project,
					Origins: o.Origins,
				},
			),
		); err != nil {
			return fmt.Errorf("query freight: %w", err)
		}

		// We didn't specify any groupBy, so there should be one group with an
		// empty key
		freight := resp.Msg.GetGroups()[""]
		return printObjects(freight.Freight, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	res := make([]*kargoapi.Freight, 0, len(o.Names)+len(o.Aliases))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetFreightResponse]
		if resp, err = kargoSvcCli.GetFreight(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetFreightRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			errs = append(errs, fmt.Errorf("get freight %s: %w", name, err))
			continue
		}
		res = append(res, resp.Msg.GetFreight())
	}
	for _, alias := range o.Aliases {
		var resp *connect.Response[v1alpha1.GetFreightResponse]
		if resp, err = kargoSvcCli.GetFreight(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetFreightRequest{
					Project: o.Project,
					Alias:   alias,
				},
			),
		); err != nil {
			errs = append(errs, fmt.Errorf("get freight %s: %w", alias, err))
			continue
		}
		res = append(res, resp.Msg.GetFreight())
	}

	if err = printObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print freight: %w", err)
	}
	return errors.Join(errs...)
}

func newFreightTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		freight := item.Object.(*kargoapi.Freight) // nolint: forcetypeassert
		var alias string
		if freight.Labels != nil {
			alias = freight.Labels[kargoapi.AliasLabelKey]
		}
		rows[i] = metav1.TableRow{
			Cells: []any{
				freight.Name,
				alias,
				freight.Origin.String(),
				duration.HumanDuration(time.Since(freight.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Alias", Type: "string"},
			{Name: "Origin", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
