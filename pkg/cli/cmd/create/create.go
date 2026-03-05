package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/resources"
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
	cmd.AddCommand(newConfigMapCommand(cfg, streams))
	cmd.AddCommand(newGenericCredentialsCommand(cfg, streams))
	cmd.AddCommand(newRepoCredentialsCommand(cfg, streams))
	cmd.AddCommand(newProjectCommand(cfg, streams))
	cmd.AddCommand(newRoleCommand(cfg, streams))
	cmd.AddCommand(newTokenCommand(cfg, streams))

	return cmd
}

// addFlags adds the flags for the create options to the provided command.
func (o *createOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

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

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	res, err := apiClient.Resources.CreateResource(
		resources.NewCreateResourceParams().
			WithManifest(string(manifest)),
		nil,
	)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}

	createErrs := make([]error, 0, len(res.Payload.Results))
	for _, r := range res.Payload.Results {
		if r.Error != "" {
			createErrs = append(createErrs, errors.New(r.Error))
			continue
		}
		if len(r.CreatedResourceManifest) > 0 {
			// Convert map to JSON then to YAML for unmarshaling
			manifestJSON, err := json.Marshal(r.CreatedResourceManifest)
			if err != nil {
				_, _ = fmt.Fprintf(o.ErrOut, "Error: %s",
					fmt.Errorf("marshal created manifest: %w", err))
				continue
			}
			var obj unstructured.Unstructured
			if err := json.Unmarshal(manifestJSON, &obj); err != nil {
				_, _ = fmt.Fprintf(o.ErrOut, "Error: %s",
					fmt.Errorf("unmarshal created manifest: %w", err))
				continue
			}
			_ = printer.PrintObj(&obj, o.Out)
		}
	}
	return errors.Join(createErrs...)
}
