package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cobracompletefig "github.com/withfig/autocomplete-tools/integrations/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/api"
	apiconfig "github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/cli/cmd/apply"
	"github.com/akuity/kargo/internal/cli/cmd/approve"
	cliconfigcmd "github.com/akuity/kargo/internal/cli/cmd/config"
	"github.com/akuity/kargo/internal/cli/cmd/create"
	"github.com/akuity/kargo/internal/cli/cmd/dashboard"
	"github.com/akuity/kargo/internal/cli/cmd/delete"
	"github.com/akuity/kargo/internal/cli/cmd/get"
	"github.com/akuity/kargo/internal/cli/cmd/login"
	"github.com/akuity/kargo/internal/cli/cmd/logout"
	"github.com/akuity/kargo/internal/cli/cmd/promote"
	"github.com/akuity/kargo/internal/cli/cmd/refresh"
	"github.com/akuity/kargo/internal/cli/cmd/update"
	"github.com/akuity/kargo/internal/cli/cmd/version"
	clicfg "github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

// rootState holds state used internally by the root command.
type rootState struct {
	localServerListener net.Listener
}

func NewRootCommand(
	cfg clicfg.CLIConfig,
	opt *option.Option,
	rs *rootState,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "kargo",
		DisableAutoGenTag: true,
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := buildRootContext(cmd.Context())

			if opt.UseLocalServer {
				restCfg, err := config.GetConfig()
				if err != nil {
					return errors.Wrap(err, "get REST config")
				}
				client, err :=
					kubernetes.NewClient(ctx, restCfg, kubernetes.ClientOptions{})
				if err != nil {
					return errors.Wrap(err, "error creating Kubernetes client")
				}
				l, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return errors.Wrap(err, "start local server")
				}
				rs.localServerListener = l
				srv := api.NewServer(
					apiconfig.ServerConfig{
						LocalMode: true,
					},
					client,
					client,
				)
				go srv.Serve(ctx, l) // nolint: errcheck
				opt.LocalServerAddress = fmt.Sprintf("http://%s", l.Addr())
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if rs.localServerListener != nil {
				return rs.localServerListener.Close()
			}
			return nil
		},
	}

	// Set up the IOStreams for the commands to use.
	streams := genericiooptions.IOStreams{Out: os.Stdout, ErrOut: os.Stderr, In: os.Stdin}
	cmd.SetIn(streams.In)
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)

	// Register the subcommands.
	cmd.AddCommand(apply.NewCommand(cfg, streams, opt))
	cmd.AddCommand(approve.NewCommand(cfg, opt))
	cmd.AddCommand(cliconfigcmd.NewCommand(cfg))
	cmd.AddCommand(create.NewCommand(cfg, streams, opt))
	cmd.AddCommand(delete.NewCommand(cfg, streams, opt))
	cmd.AddCommand(get.NewCommand(cfg, streams, opt))
	cmd.AddCommand(login.NewCommand(opt))
	cmd.AddCommand(logout.NewCommand())
	cmd.AddCommand(refresh.NewCommand(cfg, opt))
	cmd.AddCommand(update.NewCommand(cfg, opt))
	cmd.AddCommand(dashboard.NewCommand(cfg))
	cmd.AddCommand(promote.NewCommand(cfg, streams, opt))
	cmd.AddCommand(version.NewCommand(cfg, streams, opt))
	cmd.AddCommand(
		cobracompletefig.CreateCompletionSpecCommand(
			cobracompletefig.Opts{
				Use: "fig",
			},
		),
	)

	return cmd
}

func buildRootContext(ctx context.Context) context.Context {
	// TODO: Inject console printer or logger
	return ctx
}
