package apply

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/generated/resources"
)

type applyOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Filenames []string
	Recursive bool
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
		Example: templates.Example(`
# Apply a stage using the data in stage.yaml
kargo apply -f stage.yaml

# Apply the YAML resources in the stages directory
kargo apply -f stages/
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the apply options to the provided command.
func (o *applyOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Filenames(cmd.Flags(), &o.Filenames, "Filename or directory to use to apply the resource(s)")
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
	manifest, err := option.ReadManifests(o.Recursive, o.Filenames...)
	if err != nil {
		return fmt.Errorf("read manifests: %w", err)
	}

	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	// TODO: Current implementation of apply is not the same as `kubectl` does.
	// It actually "replaces" resource with the given file.
	// We should provide the same implementation as `kubectl` does.
	upsert := true
	res, err := apiClient.Resources.UpdateResource(
		resources.NewUpdateResourceParams().
			WithManifest(string(manifest)).
			WithUpsert(&upsert),
		nil,
	)
	if err != nil {
		return fmt.Errorf("apply resource: %w", err)
	}

	// Separate results into created, updated, and errors
	var createdRes, updatedRes []*models.CreateOrUpdateResourceResult
	var errs []error
	for _, r := range res.Payload.Results {
		if r.Error != "" {
			errs = append(errs, errors.New(r.Error))
		} else if r.CreatedResourceManifest != nil {
			createdRes = append(createdRes, r)
		} else if r.UpdatedResourceManifest != nil {
			updatedRes = append(updatedRes, r)
		}
	}

	// Print created resources
	if len(createdRes) > 0 {
		printer, printerErr := o.toPrinter("created")
		if printerErr != nil {
			return fmt.Errorf("new printer: %w", printerErr)
		}
		for _, res := range createdRes {
			obj := &unstructured.Unstructured{Object: res.CreatedResourceManifest}
			_ = printer.PrintObj(obj, o.Out)
		}
	}

	// Print updated resources
	if len(updatedRes) > 0 {
		printer, printerErr := o.toPrinter("configured")
		if printerErr != nil {
			return fmt.Errorf("new printer: %w", printerErr)
		}
		for _, res := range updatedRes {
			obj := &unstructured.Unstructured{Object: res.UpdatedResourceManifest}
			_ = printer.PrintObj(obj, o.Out)
		}
	}

	return errors.Join(errs...)
}

func (o *applyOptions) toPrinter(operation string) (printers.ResourcePrinter, error) {
	o.NamePrintFlags.Operation = operation
	return o.ToPrinter()
}
