package get

import (
	goerrors "errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newGetProjectsCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
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

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}
			resp, err := kargoSvcCli.ListProjects(ctx, connect.NewRequest(&v1alpha1.ListProjectsRequest{}))
			if err != nil {
				return errors.Wrap(err, "list projects")
			}

			names := slices.Compact(args)
			res := make([]*kargoapi.Project, 0, len(resp.Msg.GetProjects()))
			var resErr error
			if len(names) == 0 {
				res = append(res, resp.Msg.GetProjects()...)
			} else {
				projectsByName := make(map[string]*kargoapi.Project, len(resp.Msg.GetProjects()))
				for i := range resp.Msg.GetProjects() {
					p := resp.Msg.GetProjects()[i]
					projectsByName[p.Name] = p
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

func newProjectTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		project := item.Object.(*kargoapi.Project) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				project.Name,
				project.Status.Phase,
				duration.HumanDuration(time.Since(project.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Phase", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
