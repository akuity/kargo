package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cobracompletefig "github.com/withfig/autocomplete-tools/integrations/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/akuity/kargo/internal/api"
	apiconfig "github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/cli/apply"
	"github.com/akuity/kargo/internal/cli/create"
	"github.com/akuity/kargo/internal/cli/delete"
	"github.com/akuity/kargo/internal/cli/get"
	"github.com/akuity/kargo/internal/cli/login"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/refresh"
	"github.com/akuity/kargo/internal/cli/stage"
)

// rootState holds state used internally by the root command.
type rootState struct {
	localServerListener net.Listener
}

func NewRootCommand(opt *option.Option, rs *rootState) (*cobra.Command, error) {
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

	opt.IOStreams = &genericclioptions.IOStreams{
		In:     cmd.InOrStdin(),
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	scheme, err := option.NewScheme()
	if err != nil {
		return nil, err
	}
	opt.PrintFlags = genericclioptions.NewPrintFlags("").WithTypeSetter(scheme)
	option.InsecureTLS(&opt.InsecureTLS)(cmd.PersistentFlags())
	option.LocalServer(&opt.UseLocalServer)(cmd.PersistentFlags())

	cmd.AddCommand(apply.NewCommand(opt))
	cmd.AddCommand(create.NewCommand(opt))
	cmd.AddCommand(delete.NewCommand(opt))
	cmd.AddCommand(get.NewCommand(opt))
	cmd.AddCommand(login.NewCommand(opt))
	cmd.AddCommand(stage.NewCommand(opt))
	cmd.AddCommand(refresh.NewCommand(opt))
	cmd.AddCommand(newVersionCommand(opt))
	cmd.AddCommand(
		cobracompletefig.CreateCompletionSpecCommand(
			cobracompletefig.Opts{
				Use: "fig",
			},
		),
	)
	return cmd, nil
}

func buildRootContext(ctx context.Context) context.Context {
	// TODO: Inject console printer or logger
	return ctx
}
