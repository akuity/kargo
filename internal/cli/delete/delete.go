package delete

import (
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
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
			if len(flag.Filenames) == 0 {
				return pkgerrors.New("filename is required")
			}

			manifest, err := option.ReadManifests(flag.Filenames...)
			if err != nil {
				return pkgerrors.Wrap(err, "read manifests")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return pkgerrors.Wrap(err, "get client from config")
			}

			resp, err := kargoSvcCli.DeleteResource(ctx, connect.NewRequest(&kargosvcapi.DeleteResourceRequest{
				Manifest: manifest,
			}))
			if err != nil {
				return pkgerrors.Wrap(err, "delete resource")
			}

			var successRes []*kargosvcapi.DeleteResourceResult_DeletedResourceManifest
			var deleteErrs []error
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
						pkgerrors.Wrap(err, "Error: unmarshal deleted manifest"))
					continue
				}
				name := strings.TrimLeft(types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}.String(), "/")
				fmt.Fprintf(opt.IOStreams.Out, "%s Deleted: %q\n", obj.GetKind(), name)
			}
			return errors.Join(deleteErrs...)
		},
	}
	option.Filenames("delete", &flag.Filenames)(cmd.Flags())

	// Subcommands
	cmd.AddCommand(newProjectCommand(opt))
	cmd.AddCommand(newStageCommand(opt))
	return cmd
}
