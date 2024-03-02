package get

import (
	"context"
	goerrors "errors"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type getProjectsOptions struct {
	*option.Option
	Config config.CLIConfig

	Names []string
}

func newGetProjectsCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &getProjectsOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:     "projects [NAME ...]",
		Aliases: []string{"project"},
		Short:   "Display one or many projects",
		Example: `
# List all projects
kargo get projects

# List all projects in JSON output format
kargo get projects -o json

# Get a single project by name
kargo get project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the get projects options to the provided command.
func (o *getProjectsOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)
}

// complete sets the options from the command arguments.
func (o *getProjectsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// run gets the projects from the server and prints them to the console.
func (o *getProjectsOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	if len(o.Names) == 0 {

		var resp *connect.Response[v1alpha1.ListProjectsResponse]
		if resp, err = kargoSvcCli.ListProjects(
			ctx,
			connect.NewRequest(&v1alpha1.ListProjectsRequest{}),
		); err != nil {
			return errors.Wrap(err, "list projects")
		}
		res := make([]*kargoapi.Project, 0, len(resp.Msg.GetProjects()))
		for _, project := range resp.Msg.GetProjects() {
			res = append(res, typesv1alpha1.FromProjectProto(project))
		}
		return printObjects(o.Option, res)

	}

	res := make([]*kargoapi.Project, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetProjectResponse]
		if resp, err = kargoSvcCli.GetProject(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetProjectRequest{
					Name: name,
				},
			),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, typesv1alpha1.FromProjectProto(resp.Msg.GetProject()))
	}

	if err = printObjects(o.Option, res); err != nil {
		return errors.Wrap(err, "print credentials")
	}

	if len(errs) == 0 {
		return nil
	}

	return goerrors.Join(errs...)
}

func newProjectTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		project := item.Object.(*kargoapi.Project) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				project.Name,
				project.Status.Phase,
				duration.HumanDuration(time.Since(project.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Phase", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
