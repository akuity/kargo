package create

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	sigyaml "sigs.k8s.io/yaml"

	kargosvcapi "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type createOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Filenames []string
	Recursive bool
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "create -f FILENAME",
		Short: "Create a resource from a file or from stdin",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Create a stage using the data in stage.yaml
kargo create -f stage.yaml

# Create a project
kargo create project my-project
`),
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
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	// Register subcommands.
	cmd.AddCommand(newCredentialsCommand(cfg, streams))
	cmd.AddCommand(newProjectCommand(cfg, streams))
	cmd.AddCommand(newRoleCommand(cfg, streams))

	return cmd
}

// addFlags adds the flags for the create options to the provided command.
func (o *createOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Filenames(cmd.Flags(), &o.Filenames, "Filename or directory to use to create resource(s).")
	option.Recursive(cmd.Flags(), &o.Recursive)

	if err := cmd.MarkFlagRequired(option.FilenameFlag); err != nil {
		panic(fmt.Errorf("could not mark filename flag as required: %w", err))
	}
	if err := cmd.MarkFlagFilename(option.FilenameFlag, ".yaml", ".yml"); err != nil {
		panic(fmt.Errorf("could not mark filename flag as filename: %w", err))
	}
	if err := cmd.MarkFlagDirname(option.FilenameFlag); err != nil {
		panic(fmt.Errorf("could not mark filename flag as dirname: %w", err))
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createOptions) validate() error {
	// While the filename flag is marked as required, a user could still
	// provide an empty string. This is a check to ensure that the flag is
	// not empty.
	if len(o.Filenames) == 0 {
		return errors.New("filename is required")
	}
	return nil
}

// run performs the creation of the resource(s) using the options.
func (o *createOptions) run(ctx context.Context) error {
	manifest, err := option.ReadManifests(o.Recursive, o.Filenames...)
	if err != nil {
		return fmt.Errorf("read manifests: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	resp, err := kargoSvcCli.CreateResource(ctx, connect.NewRequest(&kargosvcapi.CreateResourceRequest{
		Manifest: manifest,
	}))
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
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
			_, _ = fmt.Fprintf(o.IOStreams.ErrOut, "Error: %s",
				fmt.Errorf("unmarshal created manifest: %w", err))
			continue
		}
		_ = printer.PrintObj(&obj, o.IOStreams.Out)
	}
	return errors.Join(createErrs...)
}
