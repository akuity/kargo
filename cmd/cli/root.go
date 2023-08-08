package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	kargoAPI "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	apioption "github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/cli/login"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/project"
	"github.com/akuity/kargo/internal/cli/stage"
	"github.com/akuity/kargo/internal/kubeclient"
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

			restCfg, err := config.GetConfig()
			if err != nil {
				return errors.Wrap(err, "error loading REST config")
			}
			var kubeClient client.Client
			{
				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(err, "error adding Kubernetes core API to scheme")
				}
				if err = kargoAPI.AddToScheme(scheme); err != nil {
					return errors.Wrap(err, "error adding Kargo API to scheme")
				}
				if kubeClient, err = client.New(
					restCfg,
					client.Options{
						Scheme: scheme,
					},
				); err != nil {
					return errors.Wrap(err, "error initializing Kubernetes client")
				}
			}

			opt.ClientOption = apioption.NewClientOption(opt.UseLocalServer)
			if opt.UseLocalServer {
				l, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return errors.Wrap(err, "start local server")
				}
				rs.localServerListener = l
				srv, err := api.NewServer(kubeClient, api.ServerConfig{})
				if err != nil {
					return errors.Wrap(err, "new api server")
				}
				go srv.Serve(ctx, l, true) // nolint: errcheck
				opt.ServerURL = fmt.Sprintf("http://%s", l.Addr())
			} else {
				cred, err := kubeclient.GetCredential(ctx, restCfg)
				if err != nil {
					return errors.Wrap(err, "get credential")
				}
				cmd.SetContext(kubeclient.SetCredentialToContext(ctx, cred))
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
	option.ServerURL(&opt.ServerURL)(cmd.PersistentFlags())
	option.LocalServer(&opt.UseLocalServer)(cmd.PersistentFlags())

	cmd.AddCommand(login.NewCommand())
	cmd.AddCommand(stage.NewCommand(opt))
	cmd.AddCommand(project.NewCommand(opt))
	cmd.AddCommand(newVersionCommand())
	return cmd, nil
}

func buildRootContext(ctx context.Context) context.Context {
	// TODO: Inject console printer or logger
	return ctx
}
