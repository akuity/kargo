package project

import (
	"net/http"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func newListCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Example: "kargo project list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client := svcv1alpha1connect.NewKargoServiceClient(http.DefaultClient, opt.ServerURL, opt.ClientOption)
			res, err := client.ListProjects(ctx, connect.NewRequest(&v1alpha1.ListProjectsRequest{
				/* explicitly empty */
			}))
			if err != nil {
				return errors.Wrap(err, "list projects")
			}
			list := &unstructured.UnstructuredList{}
			list.SetAPIVersion(metav1.Unversioned.String())
			list.SetKind("List")
			for _, project := range res.Msg.GetProjects() {
				item := &unstructured.Unstructured{}
				item.SetAPIVersion(kubev1alpha1.GroupVersion.String())
				item.SetKind("Project")
				item.SetName(project.GetName())
				list.Items = append(list.Items, *item)
			}
			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
				_ = printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(list, opt.IOStreams.Out)
				return nil
			}
			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			err = printer.PrintObj(list, opt.IOStreams.Out)
			return err
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
