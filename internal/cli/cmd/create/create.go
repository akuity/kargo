package create

import (
	"context"
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

type createOptions struct {
	*option.Option
	Config config.CLIConfig

	Filenames []string
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &createOptions{
		Option: opt,
		Config: cfg,
	}

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
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Register subcommands.
	cmd.AddCommand(newProjectCommand(cfg, opt))

	return cmd
}

// addFlags adds the flags for the create options to the provided command.
func (o *createOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

	option.Filenames(cmd.Flags(), &o.Filenames, "Filename or directory to use to create resource(s).")

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
	manifest, err := yaml.Read(o.Filenames)
	if err != nil {
		return errors.Wrap(err, "read manifests")
	}

	var printer printers.ResourcePrinter
	if ptr.Deref(o.PrintFlags.OutputFormat, "") != "" {
		printer, err = o.PrintFlags.ToPrinter()
		if err != nil {
			return errors.Wrap(err, "new printer")
		}
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
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
			fmt.Fprintf(o.IOStreams.ErrOut, "%s",
				errors.Wrap(err, "Error: unmarshal created manifest"))
			continue
		}
		if printer == nil {
			name := types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			}.String()
			fmt.Fprintf(o.IOStreams.Out, "%s Created: %q\n", obj.GetKind(), name)
			continue
		}
		_ = printer.PrintObj(&obj, o.IOStreams.Out)
	}
	return goerrors.Join(createErrs...)
}
