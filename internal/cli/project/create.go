package project

import (
	"fmt"
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newCreateCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Args:    cobra.ExactArgs(1),
		Example: "kargo project create (NAME)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("name is required")
			}

			client, err := client.GetClientFromConfig(opt)
			if err != nil {
				return err
			}

			res, err := client.CreateProject(ctx, connect.NewRequest(&v1alpha1.CreateProjectRequest{
				Name: name,
			}))
			if err != nil {
				return errors.Wrap(err, "create project")
			}
			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
				_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Created: %q", res.Msg.GetProject().GetName())
				return nil
			}
			var project unstructured.Unstructured
			project.SetAPIVersion(kubev1alpha1.GroupVersion.String())
			project.SetKind("Project")
			project.SetCreationTimestamp(metav1.NewTime(res.Msg.GetProject().GetCreateTime().AsTime()))
			project.SetName(project.GetName())
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
