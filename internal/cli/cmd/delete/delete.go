package delete

import (
	goerrors "errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/yaml"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteOptions struct {
	*option.Option

	Filenames []string
}

// addFlags adds the flags for the delete options to the provided command.
func (o *deleteOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

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

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &deleteOptions{Option: opt}

	cmd := &cobra.Command{
		Use:   "delete [--project=project] -f (FILENAME)",
		Short: "Delete resources by resources and names",
		Example: `
# Delete a project
kargo delete project my-project

# Delete a stage
kargo delete stage --project=my-project my-stage

# Delete a stage using the data in stage.yaml
kargo delete -f stage.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			manifest, err := yaml.Read(cmdOpts.Filenames)
			if err != nil {
				return errors.Wrap(err, "read manifests")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, cmdOpts.Option)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

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
			for _, r := range successRes {
				var obj unstructured.Unstructured
				if err := sigyaml.Unmarshal(r.DeletedResourceManifest, &obj); err != nil {
					fmt.Fprintf(cmdOpts.IOStreams.ErrOut, "%s",
						errors.Wrap(err, "Error: unmarshal deleted manifest"))
					continue
				}
				name := strings.TrimLeft(types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}.String(), "/")
				fmt.Fprintf(cmdOpts.IOStreams.Out, "%s Deleted: %q\n", obj.GetKind(), name)
			}
			return goerrors.Join(deleteErrs...)
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Subcommands
	cmd.AddCommand(newProjectCommand(cfg, opt))
	cmd.AddCommand(newStageCommand(cfg, opt))
	cmd.AddCommand(newWarehouseCommand(cfg, opt))
	return cmd
}
