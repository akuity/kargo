package main

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/api/config"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/rbac"
	"github.com/akuity/kargo/internal/kubernetes/event"
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
		"GOMAXPROCS", runtime.GOMAXPROCS(0),
		"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
	)

	serverCfg := config.ServerConfigFromEnv()

	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes client REST config: %w", err)
	}
	kubeClientOptions := kubernetes.ClientOptions{}
	if serverCfg.OIDCConfig != nil {
		kubeClientOptions.GlobalServiceAccountNamespaces = serverCfg.OIDCConfig.GlobalServiceAccountNamespaces
	}
	kubeClient, err := kubernetes.NewClient(ctx, restCfg, kubeClientOptions)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client for Kargo API server: %w", err)
	}

	switch {
	case !serverCfg.RolloutsIntegrationEnabled:
		o.Logger.Info("Argo Rollouts integration is disabled")
	case !argoRolloutsExists(ctx, restCfg):
		o.Logger.Info(
			"Argo Rollouts integration was enabled, but no Argo Rollouts " +
				"CRDs were found. Proceeding without Argo Rollouts integration.",
		)
		serverCfg.RolloutsIntegrationEnabled = false
	default:
		o.Logger.Info("Argo Rollouts integration is enabled")
	}

	if serverCfg.AdminConfig != nil {
		o.Logger.Info("admin account is enabled")
	}
	if serverCfg.OIDCConfig != nil {
		o.Logger.Info(
			"SSO via OpenID Connect is enabled",
			"issuerURL", serverCfg.OIDCConfig.IssuerURL,
			"clientID", serverCfg.OIDCConfig.ClientID,
			"cliClientID", serverCfg.OIDCConfig.CLIClientID,
		)
	}

	srv := api.NewServer(
		serverCfg,
		kubeClient,
		rbac.NewKubernetesRolesDatabase(kubeClient),
		event.NewRecorder(
			ctx,
			kubeClient.InternalClient().Scheme(),
			kubeClient.InternalClient(),
			"api",
		),
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
