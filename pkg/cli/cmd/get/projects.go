package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/conditions"
)

type getProjectsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Names []string
}

func newGetProjectsCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getProjectsOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "projects [NAME ...] [--no-headers]",
		Aliases: []string{"project"},
		Short:   "Display one or many projects",
		Example: templates.Example(`
# List all projects
kargo get projects

# List all projects in JSON output format
kargo get projects -o json

# Get a single project by name
kargo get project my-project
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the get projects options to the provided command.
func (o *getProjectsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)
}

// complete sets the options from the command arguments.
func (o *getProjectsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// run gets the projects from the server and prints them to the console.
func (o *getProjectsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var res *core.ListProjectsOK
		if res, err = apiClient.Core.ListProjects(
			core.NewListProjectsParams(),
			nil,
		); err != nil {
			return fmt.Errorf("list projects: %w", err)
		}
		var projectsJSON []byte
		if projectsJSON, err = json.Marshal(res.Payload); err != nil {
			return err
		}
		projectsList := struct {
			Items []*kargoapi.Project `json:"items"`
		}{}
		if err = json.Unmarshal(projectsJSON, &projectsList); err != nil {
			return err
		}
		return PrintObjects(projectsList.Items, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	projects := make([]*kargoapi.Project, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var res *core.GetProjectOK
		if res, err = apiClient.Core.GetProject(
			core.NewGetProjectParams().WithProject(name),
			nil,
		); err != nil {
			errs = append(errs, err)
			continue
		}
		var projectJSON []byte
		if projectJSON, err = json.Marshal(res.Payload); err != nil {
			errs = append(errs, err)
			continue
		}
		var project *kargoapi.Project
		if err = json.Unmarshal(projectJSON, &project); err != nil {
			errs = append(errs, err)
			continue
		}
		projects = append(projects, project)
	}

	if err = PrintObjects(projects, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print projects: %w", err)
	}
	return errors.Join(errs...)
}

func newProjectTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		project := item.Object.(*kargoapi.Project) // nolint: forcetypeassert

		var ready, status = string(metav1.ConditionUnknown), ""
		if readyCond := conditions.Get(&project.Status, kargoapi.ConditionTypeReady); readyCond != nil {
			ready = string(readyCond.Status)
			status = readyCond.Message
		}

		rows[i] = metav1.TableRow{
			Cells: []any{
				project.Name,
				ready,
				status,
				duration.HumanDuration(time.Since(project.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Ready", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
