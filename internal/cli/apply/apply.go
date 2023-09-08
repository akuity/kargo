package apply

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type Flags struct {
	Filenames []string
}

func NewCommand(opt *option.Option) *cobra.Command {
	var flag Flags
	cmd := &cobra.Command{
		Use:   "apply [--project=project] -f (FILENAME)",
		Short: "Apply a resource from a file or from stdin",
		Example: `
# Apply a stage using the data in stage.yaml
kargo apply -f stage.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(flag.Filenames) == 0 {
				return pkgerrors.New("filename is required")
			}

			rawManifest, err := option.ReadManifests(flag.Filenames...)
			if err != nil {
				return pkgerrors.Wrap(err, "read manifests")
			}

			var printer printers.ResourcePrinter
			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") != "" {
				printer, err = opt.PrintFlags.ToPrinter()
				if err != nil {
					return pkgerrors.Wrap(err, "new printer")
				}
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return pkgerrors.Wrap(err, "get client from config")
			}

			// TODO: Current implementation of apply is not the same as `kubectl` does.
			// It actually "replaces" resource with the given file.
			// We should provide the same implementation as `kubectl` does.
			resp, err := kargoSvcCli.UpdateResource(ctx,
				connect.NewRequest(&kargosvcapi.UpdateResourceRequest{
					Manifest: rawManifest,
				}))
			if err != nil {
				return pkgerrors.Wrap(err, "apply resource")
			}

			var successRes []*kargosvcapi.UpdateResourceResult_UpdatedResourceManifest
			var updateErrs []error
			for _, r := range resp.Msg.GetResults() {
				switch typedRes := r.GetResult().(type) {
				case *kargosvcapi.UpdateResourceResult_UpdatedResourceManifest:
					successRes = append(successRes, typedRes)
				case *kargosvcapi.UpdateResourceResult_Error:
					updateErrs = append(updateErrs, errors.New(typedRes.Error))
				}
			}
			for _, r := range successRes {
				var obj unstructured.Unstructured
				if err := sigyaml.Unmarshal(r.UpdatedResourceManifest, &obj); err != nil {
					fmt.Fprintf(opt.IOStreams.ErrOut, "%s",
						pkgerrors.Wrap(err, "Error: unmarshal applied manifest"))
					continue
				}
				if printer == nil {
					name := types.NamespacedName{
						Namespace: obj.GetNamespace(),
						Name:      obj.GetName(),
					}.String()
					fmt.Fprintf(opt.IOStreams.Out, "%s Applied: %q\n", obj.GetKind(), name)
					continue
				}
				_ = printer.PrintObj(&obj, opt.IOStreams.Out)
			}
			return errors.Join(updateErrs...)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Filenames("apply", &flag.Filenames)(cmd.Flags())
	return cmd
}
