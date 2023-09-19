package get

import (
	goerrors "errors"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newGetProjectsCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects [NAME...]",
		Aliases: []string{"project"},
		Short:   "Display one or many projects",
		Example: `
# List all projects
kargo get projects

# List all projects in JSON output format
kargo get projects -o json
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.New("get client from config")
			}
			resp, err := kargoSvcCli.ListProjects(ctx, connect.NewRequest(&v1alpha1.ListProjectsRequest{}))
			if err != nil {
				return errors.Wrap(err, "list projects")
			}

			names := slices.Compact(args)
			res := make([]*unstructured.Unstructured, 0, len(resp.Msg.GetProjects()))
			var resErr error
			if len(names) == 0 {
				for _, p := range resp.Msg.GetProjects() {
					res = append(res, typesv1alpha1.FromProjectProto(p))
				}
			} else {
				projectsByName := make(map[string]*unstructured.Unstructured, len(resp.Msg.GetProjects()))
				for _, p := range resp.Msg.GetProjects() {
					projectsByName[p.GetName()] = typesv1alpha1.FromProjectProto(p)
				}
				for _, name := range names {
					if promo, ok := projectsByName[name]; ok {
						res = append(res, promo)
					} else {
						resErr = goerrors.Join(err, errors.Errorf("project %q not found", name))
					}
				}
			}
			if err := printObjects(opt, res); err != nil {
				return err
			}
			return resErr
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
