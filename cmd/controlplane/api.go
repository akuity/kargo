package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

func newAPICommand() *cobra.Command {
	return &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo API Server")

			var wg sync.WaitGroup
			errCh := make(chan error, 2)

			var kubeCli client.Client
			var dynamicCli dynamic.Interface
			{
				restCfg, err := getRestConfig(ctx, os.GetEnv("KUBECONFIG", ""))
				if err != nil {
					return pkgerrors.Wrap(err, "error loading REST config")
				}
				restCfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
					return kubeclient.NewCredentialInjector(rt)
				}
				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return pkgerrors.Wrap(err, "error adding Kubernetes core API to scheme")
				}
				if err = kargoapi.AddToScheme(scheme); err != nil {
					return pkgerrors.Wrap(err, "error adding Kargo API to scheme")
				}
				mgr, err := ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme:             scheme,
						MetricsBindAddress: "0",
					},
				)
				if err != nil {
					return pkgerrors.Wrap(err, "new manager")
				}
				// Index PromotionPolicies by Stage
				if err = kubeclient.IndexPromotionPoliciesByStage(ctx, mgr); err != nil {
					return pkgerrors.Wrap(err, "index PromotionPolicies by Stage")
				}
				wg.Add(1)
				go func() {
					mgrErr := mgr.Start(ctx)
					errCh <- pkgerrors.Wrap(mgrErr, "start manager")
					wg.Done()
				}()
				kubeCli = mgr.GetClient()
				dynamicCli = dynamic.NewForConfigOrDie(restCfg)
			}

			cfg := config.ServerConfigFromEnv()

			if cfg.AdminConfig != nil {
				log.Info("admin account is enabled")
			}
			if cfg.OIDCConfig != nil {
				log.WithFields(log.Fields{
					"issuerURL": cfg.OIDCConfig.IssuerURL,
					"clientID":  cfg.OIDCConfig.ClientID,
				}).Info("SSO via OpenID Connect is enabled")
			}

			srv, err := api.NewServer(cfg, kubeCli, dynamicCli)
			if err != nil {
				return pkgerrors.Wrap(err, "error creating API server")
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
				return pkgerrors.Wrap(err, "error creating listener")
			}
			defer func() {
				_ = l.Close()
			}()
			wg.Add(1)
			go func() {
				srvErr := srv.Serve(ctx, l, false)
				errCh <- pkgerrors.Wrap(srvErr, "serve")
				wg.Done()
			}()
			wg.Wait()
			close(errCh)
			var resErr error
			for err := range errCh {
				resErr = errors.Join(resErr, err)
			}
			return resErr
		},
	}
}
