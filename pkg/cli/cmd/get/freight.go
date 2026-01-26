package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
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
	o.AddFlags(cmd)

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
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 && len(o.Aliases) == 0 {
		params := core.NewQueryFreightsRestParams().
			WithProject(o.Project)
		if len(o.Origins) > 0 {
			params = params.WithOrigins(o.Origins)
		}
		var res *core.QueryFreightsRestOK
		if res, err = apiClient.Core.QueryFreightsRest(params, nil); err != nil {
			return fmt.Errorf("query freight: %w", err)
		}
		var freightJSON []byte
		if freightJSON, err = json.Marshal(res.Payload); err != nil {
			return fmt.Errorf("marshal freight: %w", err)
		}
		// The response is {"groups": {"": {"items": [...]}}}
		type freightList struct {
			Items []*kargoapi.Freight `json:"items"`
		}
		var result struct {
			Groups map[string]*freightList `json:"groups"`
		}
		if err = json.Unmarshal(freightJSON, &result); err != nil {
			return fmt.Errorf("unmarshal freight: %w", err)
		}
		// We didn't specify any groupBy, so there should be one group with an
		// empty key
		group := result.Groups[""]
		if group == nil || len(group.Items) == 0 {
			return PrintObjects([]*kargoapi.Freight{}, o.PrintFlags, o.IOStreams, o.NoHeaders)
		}
		return PrintObjects(group.Items, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	freight := make([]*kargoapi.Freight, 0, len(o.Names)+len(o.Aliases))
	errs := make([]error, 0, len(o.Names)+len(o.Aliases))
	for _, nameOrAlias := range append(o.Names, o.Aliases...) {
		var res *core.GetFreightOK
		if res, err = apiClient.Core.GetFreight(
			core.NewGetFreightParams().
				WithProject(o.Project).
				WithFreightNameOrAlias(nameOrAlias),
			nil,
		); err != nil {
			errs = append(errs, fmt.Errorf("get freight %s: %w", nameOrAlias, err))
			continue
		}
		var freightJSON []byte
		if freightJSON, err = json.Marshal(res.Payload); err != nil {
			errs = append(errs, fmt.Errorf("marshal freight %s: %w", nameOrAlias, err))
			continue
		}
		var f *kargoapi.Freight
		if err = json.Unmarshal(freightJSON, &f); err != nil {
			errs = append(errs, fmt.Errorf("unmarshal freight %s: %w", nameOrAlias, err))
			continue
		}
		freight = append(freight, f)
	}

	if err = PrintObjects(freight, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print freight: %w", err)
	}
	return errors.Join(errs...)
}

func newFreightTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		frt := item.Object.(*kargoapi.Freight) // nolint: forcetypeassert
		var alias string
		if frt.Labels != nil {
			alias = frt.Labels[kargoapi.LabelKeyAlias]
		}
		rows[i] = metav1.TableRow{
			Cells: []any{
				frt.Name,
				alias,
				frt.Origin.String(),
				duration.HumanDuration(time.Since(frt.CreationTimestamp.Time)),
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
