package delete

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
	sigyaml "sigs.k8s.io/yaml"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/yaml"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Filenames []string
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "delete (-f FILENAME | TYPE (NAME ...))",
		Short: "Delete resources by file and names",
		Args:  option.NoArgs,
		Example: `
# Delete a stage using the data in stage.yaml
kargo delete -f stage.yaml

# Delete the YAML resources in the stages directory
kargo delete -f stages/

# Delete a project
kargo delete project my-project

# Delete a stage
kargo delete stage --project=my-project my-stage

# Delete a warehouse
kargo delete warehouse --project=my-project my-warehouse
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	cmd.SetIn(cmdOpts.IOStreams.In)
	cmd.SetOut(cmdOpts.IOStreams.Out)
	cmd.SetErr(cmdOpts.IOStreams.ErrOut)

	// Register subcommands.
	cmd.AddCommand(newProjectCommand(cfg, streams))
	cmd.AddCommand(newStageCommand(cfg, streams))
	cmd.AddCommand(newWarehouseCommand(cfg, streams))

	return cmd
}

// addFlags adds the flags for the delete options to the provided command.
func (o *deleteOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Filenames(cmd.Flags(), &o.Filenames, "Filename or directory to use to delete resource(s).")

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
func (o *deleteOptions) validate() error {
	// While the filename flag is marked as required, a user could still
	// provide an empty string. This is a check to ensure that the flag is
	// not empty.
	if len(o.Filenames) == 0 {
		return errors.New("filename is required")
	}
	return nil
}

// run performs the delete operation using the options provided.
func (o *deleteOptions) run(ctx context.Context) error {
	manifest, err := yaml.Read(o.Filenames)
	if err != nil {
		return errors.Wrap(err, "read manifests")
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}
	defer client.CloseIfPossible(kargoSvcCli)

	resp, err := kargoSvcCli.DeleteResource(ctx, connect.NewRequest(&kargosvcapi.DeleteResourceRequest{
		Manifest: manifest,
	}))
	if err != nil {
		return errors.Wrap(err, "delete resource")
	}

	resCap := len(resp.Msg.GetResults())
	successRes := make([]*kargosvcapi.DeleteResourceResult_DeletedResourceManifest, 0, resCap)
	deleteErrs := make([]error, 0, resCap)
	for _, r := range resp.Msg.GetResults() {
		switch typedRes := r.GetResult().(type) {
		case *kargosvcapi.DeleteResourceResult_DeletedResourceManifest:
			successRes = append(successRes, typedRes)
		case *kargosvcapi.DeleteResourceResult_Error:
			deleteErrs = append(deleteErrs, errors.New(typedRes.Error))
		}
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "create printer")
	}

	for _, r := range successRes {
		var obj unstructured.Unstructured
		if err := sigyaml.Unmarshal(r.DeletedResourceManifest, &obj); err != nil {
			fmt.Fprintf(o.IOStreams.ErrOut, "%s",
				errors.Wrap(err, "Error: unmarshal deleted manifest"))
			continue
		}
		_ = printer.PrintObj(&obj, o.IOStreams.Out)
	}
	return goerrors.Join(deleteErrs...)
}
