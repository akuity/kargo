package main

import (
	"context"
	"fmt"
	"net"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/rbac"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/os"
	versionpkg "github.com/akuity/kargo/internal/version"
)

type apiOptions struct {
	KubeConfig string

	Host string
	Port string

	Logger *logging.Logger
}

func newAPICommand() *cobra.Command {
	cmdOpts := &apiOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLogger(logging.InfoLevel),
	}

	cmd := &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdOpts.complete()

			return cmdOpts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *apiOptions) complete() {
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")

	o.Host = os.GetEnv("HOST", "0.0.0.0")
	o.Port = os.GetEnv("PORT", "8080")
}

func (o *apiOptions) run(ctx context.Context) error {
	version := versionpkg.GetVersion()
	o.Logger.Info(
		"Starting Kargo API Server",
		"version", version.Version,
		"commit", version.GitCommit,
	)

	cfg := config.ServerConfigFromEnv()

	clientCfg, internalClient, recorder, err := o.setupAPIClient(ctx)
	if err != nil {
		return fmt.Errorf("error setting up internal Kubernetes API client: %w", err)
	}

	kubeClient, err := newWrappedKubernetesClient(ctx, clientCfg, internalClient, cfg)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client for Kargo API server: %w", err)
	}
	switch {
	case !cfg.RolloutsIntegrationEnabled:
		o.Logger.Info("Argo Rollouts integration is disabled")
	case !argoRolloutsExists(ctx, clientCfg):
		o.Logger.Info(
			"Argo Rollouts integration was enabled, but no Argo Rollouts " +
				"CRDs were found. Proceeding without Argo Rollouts integration.",
		)
		cfg.RolloutsIntegrationEnabled = false
	default:
		o.Logger.Info("Argo Rollouts integration is enabled")
	}

	if cfg.AdminConfig != nil {
		o.Logger.Info("admin account is enabled")
	}
	if cfg.OIDCConfig != nil {
		o.Logger.Info(
			"SSO via OpenID Connect is enabled",
			"issuerURL", cfg.OIDCConfig.IssuerURL,
			"clientID", cfg.OIDCConfig.ClientID,
			"cliClientID", cfg.OIDCConfig.CLIClientID,
		)
	}

	srv := api.NewServer(
		cfg,
		kubeClient,
		internalClient,
		rbac.NewKubernetesRolesDatabase(kubeClient),
		recorder,
	)
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", o.Host, o.Port))
	if err != nil {
		return fmt.Errorf("error creating listener: %w", err)
	}
	defer l.Close()

	if err = srv.Serve(ctx, l); err != nil {
		return fmt.Errorf("error serving API: %w", err)
	}
	return nil
}

func (o *apiOptions) setupAPIClient(ctx context.Context) (*rest.Config, client.Client, record.EventRecorder, error) {
	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get REST config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err = kubescheme.AddToScheme(scheme); err != nil {
		return nil, nil, nil, fmt.Errorf("error adding Kubernetes API to Kargo API manager scheme: %w", err)
	}

	if err = rbacv1.AddToScheme(scheme); err != nil {
		return nil, nil, nil, fmt.Errorf(
			"error adding Kubernetes RBAC API to Kargo controller manager scheme: %w",
			err,
		)
	}

	if err = rollouts.AddToScheme(scheme); err != nil {
		return nil, nil, nil, fmt.Errorf("error adding Argo Rollouts API to Kargo API manager scheme: %w", err)
	}

	if err = kargoapi.AddToScheme(scheme); err != nil {
		return nil, nil, nil, fmt.Errorf("error adding Kargo API to Kargo API manager scheme: %w", err)
	}

	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
				},
			},
		},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error initializing Kargo API manager: %w", err)
	}

	if err = registerKargoIndexers(ctx, mgr); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to register Kargo indexers: %w", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic(fmt.Errorf("error starting Kargo API manager: %w", err))
		}
	}()

	return restCfg, mgr.GetClient(), libEvent.NewRecorder(ctx, scheme, mgr.GetClient(), "api"), nil
}

func registerKargoIndexers(ctx context.Context, mgr ctrl.Manager) error {
	// Index Promotions by Stage
	if err := kubeclient.IndexPromotionsByStage(ctx, mgr); err != nil {
		return fmt.Errorf("index Promotions by Stage: %w", err)
	}

	// Index Freight by Warehouse
	if err := kubeclient.IndexFreightByWarehouse(ctx, mgr); err != nil {
		return fmt.Errorf("index Freight by Warehouse: %w", err)
	}

	// Index Freight by Stages in which it has been verified
	if err := kubeclient.IndexFreightByVerifiedStages(ctx, mgr); err != nil {
		return fmt.Errorf("index Freight by Stages in which it has been verified: %w", err)
	}

	// Index Freight by Stages for which it is approved
	if err := kubeclient.IndexFreightByApprovedStages(ctx, mgr); err != nil {
		return fmt.Errorf("index Freight by Stages for which it has been approved: %w", err)
	}

	// Index ServiceAccounts by OIDC Claim
	if err := kubeclient.IndexServiceAccountsByOIDCClaim(ctx, mgr); err != nil {
		return fmt.Errorf("index ServiceAccounts by OIDC claim: %w", err)
	}

	// Index Events by InvolvedObject's API Group
	if err := kubeclient.IndexEventsByInvolvedObjectAPIGroup(ctx, mgr); err != nil {
		return fmt.Errorf("index Events by InvolvedObject's API group: %w", err)
	}

	return nil
}

func newWrappedKubernetesClient(
	ctx context.Context,
	restCfg *rest.Config,
	internalClient client.Client,
	serverCfg config.ServerConfig,
) (kubernetes.Client, error) {
	kubeClientOptions := kubernetes.ClientOptions{
		NewInternalClient: func(context.Context, *rest.Config, *runtime.Scheme) (client.Client, error) {
			return internalClient, nil
		},
	}
	if serverCfg.OIDCConfig != nil {
		kubeClientOptions.GlobalServiceAccountNamespaces = serverCfg.OIDCConfig.GlobalServiceAccountNamespaces
	}
	return kubernetes.NewClient(ctx, restCfg, kubeClientOptions)
}
