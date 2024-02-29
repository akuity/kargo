package create

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	sigyaml "sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type createProjectOptions struct {
	*option.Option
	Config config.CLIConfig

	Name string
}

func newProjectCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &createProjectOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "project (NAME)",
		Short: "Create a project",
		Args:  option.MinimumNArgs(1),
		Example: `
# Create project
kargo create project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the create project options to the provided command.
func (o *createProjectOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)
}

// complete sets the options from the command arguments.
func (o *createProjectOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createProjectOptions) validate() error {
	if o.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// run creates a project using the provided options.
func (o *createProjectOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	project := &kargoapi.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: o.Name,
		},
	}
	projectBytes, err := sigyaml.Marshal(project)
	if err != nil {
		return errors.Wrap(err, "marshal project")
	}

	resp, err := kargoSvcCli.CreateResource(
		ctx,
		connect.NewRequest(
			&kargosvcapi.CreateResourceRequest{
				Manifest: projectBytes,
			},
		),
	)
	if err != nil {
		return errors.Wrap(err, "create resource")
	}

	project = &kargoapi.Project{}
	projectBytes = resp.Msg.GetResults()[0].GetCreatedResourceManifest()
	if err = sigyaml.Unmarshal(projectBytes, project); err != nil {
		return errors.Wrap(err, "unmarshal project")
	}

	if ptr.Deref(o.PrintFlags.OutputFormat, "") == "" {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "Project Created: %q\n", o.Name)
		return nil
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "new printer")
	}
	return printer.PrintObj(project, o.IOStreams.Out)
}
