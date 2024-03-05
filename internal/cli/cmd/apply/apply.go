package apply

import (
	"context"
	goerrors "errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/yaml"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type applyOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Filenames []string
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &applyOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME",
		Short: "Apply a resource from a file or from stdin",
		Args:  option.NoArgs,
		Example: `
# Apply a stage using the data in stage.yaml
kargo apply -f stage.yaml

# Apply the YAML resources in the stages directory
kargo apply -f stages/
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Set the input/output streams for the command.
	cmd.SetIn(cmdOpts.IOStreams.In)
	cmd.SetOut(cmdOpts.IOStreams.Out)
	cmd.SetErr(cmdOpts.IOStreams.ErrOut)

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the apply options to the provided command.
func (o *applyOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

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

// run performs the apply operation using the provided options.
func (o *applyOptions) run(ctx context.Context) error {
	rawManifest, err := yaml.Read(o.Filenames)
	if err != nil {
		return errors.Wrap(err, "read manifests")
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}
	defer client.CloseIfPossible(kargoSvcCli)

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

	printer, err := o.toPrinter("created")
	if err != nil {
		return errors.Wrap(err, "new printer")
	}

	for _, res := range createdRes {
		var obj unstructured.Unstructured
		if err = sigyaml.Unmarshal(res.CreatedResourceManifest, &obj); err != nil {
			fmt.Fprintf(o.IOStreams.ErrOut, "%s",
				errors.Wrap(err, "Error: unmarshal created manifest"))
			continue
		}
		_ = printer.PrintObj(&obj, o.IOStreams.Out)
	}

	printer, err = o.toPrinter("updated")
	if err != nil {
		return errors.Wrap(err, "new printer")
	}

	for _, res := range updatedRes {
		var obj unstructured.Unstructured
		if err = sigyaml.Unmarshal(res.UpdatedResourceManifest, &obj); err != nil {
			fmt.Fprintf(o.IOStreams.ErrOut, "%s",
				errors.Wrap(err, "Error: unmarshal updated manifest"))
			continue
		}
		_ = printer.PrintObj(&obj, o.IOStreams.Out)
	}
	return goerrors.Join(errs...)
}

func (o *applyOptions) toPrinter(operation string) (printers.ResourcePrinter, error) {
	o.PrintFlags.NamePrintFlags.Operation = operation
	return o.PrintFlags.ToPrinter()
}
