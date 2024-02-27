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

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	var filenames []string
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
			if len(filenames) == 0 {
				return errors.New("filename is required")
			}

			manifest, err := yaml.Read(filenames)
			if err != nil {
				return errors.Wrap(err, "read manifests")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
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
					fmt.Fprintf(opt.IOStreams.ErrOut, "%s",
						errors.Wrap(err, "Error: unmarshal deleted manifest"))
					continue
				}
				name := strings.TrimLeft(types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}.String(), "/")
				fmt.Fprintf(opt.IOStreams.Out, "%s Deleted: %q\n", obj.GetKind(), name)
			}
			return goerrors.Join(deleteErrs...)
		},
	}
	option.Filenames(cmd.Flags(), &filenames, "apply")
	option.InsecureTLS(cmd.PersistentFlags(), opt)
	option.LocalServer(cmd.PersistentFlags(), opt)

	// Subcommands
	cmd.AddCommand(newProjectCommand(cfg, opt))
	cmd.AddCommand(newStageCommand(cfg, opt))
	cmd.AddCommand(newWarehouseCommand(cfg, opt))
	return cmd
}
