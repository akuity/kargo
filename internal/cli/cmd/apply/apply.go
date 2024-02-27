package apply

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
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/yaml"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type applyOptions struct {
	*option.Option

	Filenames []string
}

// addFlags adds the flags for the apply options to the provided command.
func (o *applyOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

	option.Filenames(cmd.Flags(), &o.Filenames, "Filename or directory to use to apply the resource(s)")

	if err := cmd.MarkFlagRequired(option.FilenameFlag); err != nil {
		panic(errors.Wrap(err, "could not mark filename flag as required"))
	}
	if err := cmd.MarkFlagFilename(option.FilenameFlag, ".yaml", ".yml"); err != nil {
		panic(errors.Wrap(err, "could not mark filename flag as filename"))
	}
	if err := cmd.MarkFlagDirname(option.FilenameFlag); err != nil {
		panic(errors.Wrap(err, "could not mark filename flag as dirname"))
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *applyOptions) validate() error {
	// While the filename flag is marked as required, a user could still
	// provide an empty string. This is a check to ensure that the flag is
	// not empty.
	if len(o.Filenames) == 0 {
		return errors.New("filename is required")
	}
	return nil
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &applyOptions{Option: opt}

	cmd := &cobra.Command{
		Use:   "apply [--project=project] -f (FILENAME)",
		Short: "Apply a resource from a file or from stdin",
		Example: `
# Apply a stage using the data in stage.yaml
kargo apply -f stage.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			rawManifest, err := yaml.Read(cmdOpts.Filenames)
			if err != nil {
				return errors.Wrap(err, "read manifests")
			}

			var printer printers.ResourcePrinter
			if ptr.Deref(cmdOpts.PrintFlags.OutputFormat, "") != "" {
				printer, err = cmdOpts.PrintFlags.ToPrinter()
				if err != nil {
					return errors.Wrap(err, "new printer")
				}
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, cmdOpts.Option)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			// TODO: Current implementation of apply is not the same as `kubectl` does.
			// It actually "replaces" resource with the given file.
			// We should provide the same implementation as `kubectl` does.
			resp, err := kargoSvcCli.CreateOrUpdateResource(ctx,
				connect.NewRequest(&kargosvcapi.CreateOrUpdateResourceRequest{
					Manifest: rawManifest,
				}))
			if err != nil {
				return errors.Wrap(err, "apply resource")
			}

			resCap := len(resp.Msg.GetResults())
			createdRes := make([]*kargosvcapi.CreateOrUpdateResourceResult_CreatedResourceManifest, 0, resCap)
			updatedRes := make([]*kargosvcapi.CreateOrUpdateResourceResult_UpdatedResourceManifest, 0, resCap)
			errs := make([]error, 0, resCap)
			for _, r := range resp.Msg.GetResults() {
				switch typedRes := r.GetResult().(type) {
				case *kargosvcapi.CreateOrUpdateResourceResult_CreatedResourceManifest:
					createdRes = append(createdRes, typedRes)
				case *kargosvcapi.CreateOrUpdateResourceResult_UpdatedResourceManifest:
					updatedRes = append(updatedRes, typedRes)
				case *kargosvcapi.CreateOrUpdateResourceResult_Error:
					errs = append(errs, errors.New(typedRes.Error))
				}
			}
			for _, r := range createdRes {
				var obj unstructured.Unstructured
				if err := sigyaml.Unmarshal(r.CreatedResourceManifest, &obj); err != nil {
					fmt.Fprintf(cmdOpts.IOStreams.ErrOut, "%s",
						errors.Wrap(err, "Error: unmarshal created manifest"))
					continue
				}
				if printer == nil {
					name := types.NamespacedName{
						Namespace: obj.GetNamespace(),
						Name:      obj.GetName(),
					}.String()
					fmt.Fprintf(cmdOpts.IOStreams.Out, "%s Created: %q\n", obj.GetKind(), name)
					continue
				}
				_ = printer.PrintObj(&obj, cmdOpts.IOStreams.Out)
			}
			for _, r := range updatedRes {
				var obj unstructured.Unstructured
				if err := sigyaml.Unmarshal(r.UpdatedResourceManifest, &obj); err != nil {
					fmt.Fprintf(cmdOpts.IOStreams.ErrOut, "%s",
						errors.Wrap(err, "Error: unmarshal updated manifest"))
					continue
				}
				if printer == nil {
					name := types.NamespacedName{
						Namespace: obj.GetNamespace(),
						Name:      obj.GetName(),
					}.String()
					fmt.Fprintf(cmdOpts.IOStreams.Out, "%s Updated: %q\n", obj.GetKind(), name)
					continue
				}
				_ = printer.PrintObj(&obj, cmdOpts.IOStreams.Out)
			}
			return goerrors.Join(errs...)
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}
