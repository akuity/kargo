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

			kargoMgr, argoMgr, err := getMgrs()
			if err != nil {
				return errors.Wrap(err, "error getting controller manager")
			}

			credentialsDB, err := credentials.NewKubernetesDatabase(
				ctx,
				cfg.ArgoCDNamespace,
				kargoMgr,
				argoMgr,
			)
			if err != nil {
				return errors.Wrap(err, "error initializing credentials DB")
			}

			if err := environments.SetupReconcilerWithManager(
				ctx,
				kargoMgr,
				argoMgr,
				credentialsDB,
			); err != nil {
				return errors.Wrap(err, "error setting up Environments reconciler")
			}
			if err := promotions.SetupReconcilerWithManager(
				kargoMgr,
				argoMgr,
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
				applications.SetupReconcilerWithManager(kargoMgr, argoMgr); err != nil {
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
			if argoMgr != kargoMgr {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := argoMgr.Start(ctx); err != nil {
						errChan <- errors.Wrap(err, "error starting argo manager")
					}
				}()
			}

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

func getMgrs() (manager.Manager, manager.Manager, error) {
	kargoMgrCfg, argoMgrCfg, err := getMgrConfigs()
	if err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}

	kargoMgrScheme, argoMgrScheme, err := getMgrSchemes(kargoMgrCfg, argoMgrCfg)
	if err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}

	kargoMgr, err := ctrl.NewManager(
		kargoMgrCfg,
		ctrl.Options{
			Scheme:             kargoMgrScheme,
			MetricsBindAddress: "0",
		},
	)
	if err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}

	var argoMgr manager.Manager
	if argoMgrScheme == kargoMgrScheme {
		argoMgr = kargoMgr
	} else {
		argoMgr, err = ctrl.NewManager(
			argoMgrCfg,
			ctrl.Options{
				Scheme:             argoMgrScheme,
				MetricsBindAddress: "0",
			},
		)
		if err != nil {
			// TODO: Wrap this
			return nil, nil, err
		}
	}

	return kargoMgr, argoMgr, nil
}

func getMgrSchemes(kargoMgrCfg, argoMgrCfg *rest.Config) (*runtime.Scheme, *runtime.Scheme, error) {
	kargoMgrScheme := runtime.NewScheme()
	var argoMgrScheme *runtime.Scheme
	if argoMgrCfg == kargoMgrCfg {
		argoMgrScheme = kargoMgrScheme
	} else {
		argoMgrScheme = runtime.NewScheme()
	}

	// Schemes used by the Kargo controller manager
	if err := corev1.AddToScheme(kargoMgrScheme); err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}
	if err := rbacv1.AddToScheme(kargoMgrScheme); err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}
	if err := api.AddToScheme(kargoMgrScheme); err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}

	// Schemes used by the Argo CD controller manager
	if err := corev1.AddToScheme(argoMgrScheme); err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}
	if err := argocd.AddToScheme(argoMgrScheme); err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}

	return kargoMgrScheme, argoMgrScheme, nil
}

func getMgrConfigs() (*rest.Config, *rest.Config, error) {
	const kargoCtx = "kargo"
	const argoCtx = "argo"

	mgrCfg, err := config.GetConfig()
	if err != nil {
		// TODO: Wrap this
		return nil, nil, err
	}

	kargoMgrCfg, err := config.GetConfigWithContext(kargoCtx)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			kargoMgrCfg = mgrCfg
		} else {
			// TODO: Wrap this
			return nil, nil, err
		}
	}

	argoMgrCfg, err := config.GetConfigWithContext(argoCtx)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			argoMgrCfg = mgrCfg
		} else {
			// TODO: Wrap this
			return nil, nil, err
		}
	}

	return kargoMgrCfg, argoMgrCfg, nil
}
