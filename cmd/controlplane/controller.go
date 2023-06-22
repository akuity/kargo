package main

import (
	"strings"
	"sync"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	libConfig "github.com/akuity/kargo/internal/config"
	"github.com/akuity/kargo/internal/controller/applications"
	"github.com/akuity/kargo/internal/controller/environments"
	"github.com/akuity/kargo/internal/controller/promotions"
	"github.com/akuity/kargo/internal/credentials"
	versionpkg "github.com/akuity/kargo/internal/version"
)

func newControllerCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "controller",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			version := versionpkg.GetVersion()
			log.WithFields(log.Fields{
				"version": version.Version,
				"commit":  version.GitCommit,
			}).Info("Starting Kargo Controller")

			cfg := libConfig.NewControllerConfig()

			var kargoMgr manager.Manager
			{
				restCfg, err := getRestConfig("kargo", false)
				if err != nil {
					return errors.Wrap(
						err,
						"error loading REST config for Kargo controller manager",
					)
				}
				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes core API to Kargo controller manager "+
							"scheme",
					)
				}
				if err = rbacv1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes RBAC API to Kargo controller manager "+
							"scheme",
					)
				}
				if err = api.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kargo API to Kargo controller manager scheme",
					)
				}
				if kargoMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme:             scheme,
						MetricsBindAddress: "0",
					},
				); err != nil {
					return errors.Wrap(err, "error initializing Kargo controller manager")
				}
			}

			var appMgr manager.Manager
			{
				restCfg, err :=
					getRestConfig("argo", cfg.ArgoCDPreferInClusterRestConfig)
				if err != nil {
					return errors.Wrap(
						err,
						"error loading REST config for Argo CD Application controller "+
							"manager",
					)
				}
				scheme := runtime.NewScheme()
				if err = corev1.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kubernetes core API to Argo CD Application "+
							"controller manager scheme",
					)
				}
				if err = argocd.AddToScheme(scheme); err != nil {
					return errors.Wrap(
						err,
						"error adding Kargo API to Argo CD Application controller manager "+
							"scheme",
					)
				}
				if appMgr, err = ctrl.NewManager(
					restCfg,
					ctrl.Options{
						Scheme:             scheme,
						MetricsBindAddress: "0",
					},
				); err != nil {
					return errors.Wrap(
						err,
						"error initializing Argo CD Application controller manager",
					)
				}
			}

			argoMgrForCreds := appMgr
			if !cfg.ArgoCDCredentialBorrowingEnabled {
				argoMgrForCreds = nil
			}
			credentialsDB, err := credentials.NewKubernetesDatabase(
				ctx,
				cfg.ArgoCDNamespace,
				kargoMgr,
				argoMgrForCreds,
			)
			if err != nil {
				return errors.Wrap(err, "error initializing credentials DB")
			}

			if err := environments.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				appMgr,
				credentialsDB,
			); err != nil {
				return errors.Wrap(err, "error setting up Environments reconciler")
			}

			if err := promotions.SetupReconcilerWithManager(
				kargoMgr,
				appMgr,
				credentialsDB,
				bookkeeper.NewService(
					&bookkeeper.ServiceOptions{
						LogLevel: bookkeeper.LogLevel(cfg.LogLevel),
					},
				),
			); err != nil {
				return errors.Wrap(err, "error setting up Promotions reconciler")
			}

			if err :=
				applications.SetupReconcilerWithManager(kargoMgr, appMgr); err != nil {
				return errors.Wrap(err, "error setting up Applications reconciler")
			}

			var errChan = make(chan error)

			wg := sync.WaitGroup{}

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := kargoMgr.Start(ctx); err != nil {
					errChan <- errors.Wrap(err, "error starting kargo manager")
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := appMgr.Start(ctx); err != nil {
					errChan <- errors.Wrap(err, "error starting argo manager")
				}
			}()

			// Adapt wg to a channel that can be used in a select
			doneCh := make(chan struct{})
			go func() {
				wg.Wait()
				close(doneCh)
			}()

			select {
			case err := <-errChan:
				return err
			case <-doneCh:
				return nil
			}
		},
	}
}

func getRestConfig(
	cfgCtx string,
	preferInClusterCfg bool,
) (*rest.Config, error) {
	var cfg *rest.Config
	var err error
	if preferInClusterCfg {
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, errors.Wrapf(err, "error loading in-cluster rest config")
		}
		return cfg, nil
	}
	if cfg, err = config.GetConfigWithContext(cfgCtx); err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			if cfg, err = rest.InClusterConfig(); err != nil {
				return nil, errors.Wrapf(err, "error loading default rest config")
			}
			return cfg, nil
		}
		return nil,
			errors.Wrapf(err, "error loading rest config for context %q", cfgCtx)
	}
	return cfg, nil
}
