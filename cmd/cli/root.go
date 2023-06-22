package main

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	kargoAPI "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	apioption "github.com/akuity/kargo/internal/api/option"
	"github.com/akuity/kargo/internal/cli/env"
	"github.com/akuity/kargo/internal/cli/option"
	libConfig "github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/kubeclient"
)

type localServerListenerKey struct {
	// explicitly empty
}

func NewRootCommand(opt *option.Option) *cobra.Command {
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
				ctx = context.WithValue(ctx, localServerListenerKey{}, l)
				srv, err := api.NewServer(
					kubeClient,
					libConfig.APIConfig{
						LocalMode: true,
					},
				)
				if err != nil {
					return errors.Wrap(err, "new api server")
				}
				go func() {
					_ = srv.Serve(ctx, l)
				}()
				opt.ServerURL = fmt.Sprintf("http://%s", l.Addr())
			} else {
				cred, err := kubeclient.GetCredential(ctx, restCfg)
				if err != nil {
					return errors.Wrap(err, "get credential")
				}
				ctx = kubeclient.SetCredentialToContext(ctx, cred)
			}
			cmd.SetContext(ctx)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if l, ok := cmd.Context().Value(localServerListenerKey{}).(net.Listener); ok {
				return l.Close()
			}
			return nil
		},
	}

	option.ServerURL(&opt.ServerURL)(cmd.PersistentFlags())
	option.LocalServer(&opt.UseLocalServer)(cmd.PersistentFlags())

	cmd.AddCommand(env.NewCommand(opt))
	cmd.AddCommand(newVersionCommand())
	return cmd
}

func buildRootContext(ctx context.Context) context.Context {
	// TODO: Inject console printer or logger
	return ctx
}
