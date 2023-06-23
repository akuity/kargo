package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	kargoAPI "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var kubeClient client.Client
			{
				restCfg, err := config.GetConfig()
				if err != nil {
					return errors.Wrap(err, "error loading REST config")
				}
				restCfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
					return kubeclient.NewCredentialInjector(rt)
				}
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

			srv, err := api.NewServer(kubeClient, api.NewServerConfig())
			if err != nil {
				return errors.Wrap(err, "error creating API server")
			}
			l, err := net.Listen(
				"tcp",
				fmt.Sprintf(
					"%s:%s",
					os.GetEnv("HOST", "0.0.0.0"),
					os.GetEnv("PORT", "8080"),
				),
			)
			if err != nil {
				return errors.Wrap(err, "error creating listener")
			}
			defer l.Close()
			return srv.Serve(cmd.Context(), l, false)
		},
	}
}
