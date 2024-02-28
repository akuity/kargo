package create

import (
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

func newProjectCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
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

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			project := &kargoapi.Project{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kargoapi.GroupVersion.String(),
					Kind:       "Project",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
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

			if ptr.Deref(opt.PrintFlags.OutputFormat, "") == "" {
				_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Created: %q\n", name)
				return nil
			}
			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			return printer.PrintObj(project, opt.IOStreams.Out)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
