package create

import (
	goerrors "errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/ptr"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func NewCommand(opt *option.Option) *cobra.Command {
	var filenames []string
	cmd := &cobra.Command{
		Use:   "create [--project=project] -f (FILENAME)",
		Short: "Create a resource from a file or from stdin",
		Example: `
# Create a stage using the data in stage.yaml
kargo create -f stage.yaml

# Create project
kargo create project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(filenames) == 0 {
				return errors.New("filename is required")
			}

			manifest, err := option.ReadManifests(filenames...)
			if err != nil {
				return errors.Wrap(err, "read manifests")
			}

			var printer printers.ResourcePrinter
			if ptr.Deref(opt.PrintFlags.OutputFormat, "") != "" {
				printer, err = opt.PrintFlags.ToPrinter()
				if err != nil {
					return errors.Wrap(err, "new printer")
				}
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}
			resp, err := kargoSvcCli.CreateResource(ctx, connect.NewRequest(&kargosvcapi.CreateResourceRequest{
				Manifest: manifest,
			}))
			if err != nil {
				return errors.Wrap(err, "create resource")
			}

			resCap := len(resp.Msg.GetResults())
			successRes := make([]*kargosvcapi.CreateResourceResult_CreatedResourceManifest, 0, resCap)
			createErrs := make([]error, 0, resCap)
			for _, r := range resp.Msg.GetResults() {
				switch typedRes := r.GetResult().(type) {
				case *kargosvcapi.CreateResourceResult_CreatedResourceManifest:
					successRes = append(successRes, typedRes)
				case *kargosvcapi.CreateResourceResult_Error:
					createErrs = append(createErrs, errors.New(typedRes.Error))
				}
			}
			for _, r := range successRes {
				var obj unstructured.Unstructured
				if err := sigyaml.Unmarshal(r.CreatedResourceManifest, &obj); err != nil {
					fmt.Fprintf(opt.IOStreams.ErrOut, "%s",
						errors.Wrap(err, "Error: unmarshal created manifest"))
					continue
				}
				if printer == nil {
					name := types.NamespacedName{
						Namespace: obj.GetNamespace(),
						Name:      obj.GetName(),
					}.String()
					fmt.Fprintf(opt.IOStreams.Out, "%s Created: %q\n", obj.GetKind(), name)
					continue
				}
				_ = printer.PrintObj(&obj, opt.IOStreams.Out)
			}
			return goerrors.Join(createErrs...)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Filenames(cmd.Flags(), &filenames, "apply")

	// Subcommands
	cmd.AddCommand(newProjectCommand(opt))
	return cmd
}
