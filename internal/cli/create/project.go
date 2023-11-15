package create

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newProjectCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project (NAME)",
		Short: "Create a project",
		Args:  option.MinimumNArgs(1),
		Example: `
# Create project
kargo create project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("name is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.New("get client from config")
			}
			resp, err := kargoSvcCli.CreateProject(ctx,
				connect.NewRequest(&kargosvcapi.CreateProjectRequest{
					Name: name,
				}))
			if err != nil {
				return errors.Wrap(err, "create project")
			}

			var project unstructured.Unstructured
			project.SetAPIVersion(kargoapi.GroupVersion.String())
			project.SetKind("Project")
			project.SetCreationTimestamp(metav1.NewTime(resp.Msg.GetProject().GetCreateTime().AsTime()))
			project.SetName(resp.Msg.GetProject().GetName())

			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
				_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Created: %q\n", name)
				return nil
			}
			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			return printer.PrintObj(&project, opt.IOStreams.Out)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
