package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	"github.com/akuity/kargo/pkg/event"
	natsevent "github.com/akuity/kargo/pkg/event/nats"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/password"
	"github.com/akuity/kargo/pkg/server"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/projectconfigs"
	"github.com/akuity/kargo/pkg/server/rbac"
	"github.com/akuity/kargo/pkg/telemetry"
	"github.com/akuity/kargo/pkg/types"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type apiOptions struct {
	Database   database.Config
	KubeConfig string
	QPS        float32
	Burst      int

	BindAddress string
	Port        string

	Logger *logging.Logger
}

func newAPICommand() *cobra.Command {
	_, format := getLogVars()
	cmdOpts := &apiOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLoggerOrDie(logging.InfoLevel, format),
	}

	cmd := &cobra.Command{
		Use:               "api",
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			version := versionpkg.GetVersion()
			cmdOpts.Logger.Info(
				"Starting Kargo API Server",
				"version", version.Version,
				"commit", version.GitCommit,
				"GOMAXPROCS", runtime.GOMAXPROCS(0),
				"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
			)
			cmdOpts.complete()

			return cmdOpts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *apiOptions) complete() {
	o.Database = database.ConfigFromEnv()
	o.KubeConfig = os.GetEnv("KUBECONFIG", "")
	o.QPS = types.MustParseFloat32(os.GetEnv("KUBE_API_QPS", "50.0"))
	o.Burst = types.MustParseInt(os.GetEnv("KUBE_API_BURST", "300"))

	o.BindAddress = os.GetEnv("BIND_ADDRESS", "0.0.0.0")
	o.Port = os.GetEnv("PORT", "8080")

	logLevel, logFormat := getLogVars()

	o.Logger = logging.NewLoggerOrDie(logLevel, logFormat)
}

func (o *apiOptions) run(ctx context.Context) error {
	// The database holds ProjectConfigs, which the API server cannot serve
	// without it.
	if !o.Database.Configured() {
		return errors.New(
			"no database is configured: the API server requires one; " +
				"set DATABASE_URL or DATABASE_HOST",
		)
	}

	shutdownTelemetry, err := telemetry.SetupFromEnv(
		logging.ContextWithLogger(ctx, o.Logger),
		"api",
	)
	if err != nil {
		return err
	}
	defer shutdownTelemetry()

	serverCfg := config.ServerConfigFromEnv()
	if serverCfg.AdminConfig != nil {
		if err = password.Validate(serverCfg.AdminConfig.HashedPassword); err != nil {
			return fmt.Errorf(
				"invalid admin account password hash (generate one with `kargo hash-password`): %w",
				err,
			)
		}
	}

	natsConn, err := connectNATS(logging.ContextWithLogger(ctx, o.Logger))
	if err != nil {
		return err
	}
	defer natsConn.Close()

	restCfg, err := kubernetes.GetRestConfig(ctx, o.KubeConfig)
	if err != nil {
		return fmt.Errorf("error getting Kubernetes client REST config: %w", err)
	}
	kubernetes.ConfigureQPSBurst(ctx, restCfg, o.QPS, o.Burst)

	kubeClientOptions := kubernetes.ClientOptions{
		KargoNamespace: serverCfg.KargoNamespace,
	}
	kubeClient, err := kubernetes.NewClient(ctx, restCfg, kubeClientOptions)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client for Kargo API server: %w", err)
	}

	if serverCfg.RolloutsIntegrationEnabled {
		var exists bool
		if exists, err = argoRolloutsExists(ctx, restCfg); !exists || err != nil {
			// If we are unable to determine if Argo Rollouts is installed, we
			// will return an error and fail to start the server. Note this
			// will only happen if we get an inconclusive response from the API
			// server (e.g. due to network issues), and not if Argo Rollouts is
			// not installed.
			if err != nil {
				return fmt.Errorf("unable to determine if Argo Rollouts is installed: %w", err)
			}

			o.Logger.Info(
				"Argo Rollouts integration was enabled, but no Argo Rollouts " +
					"CRDs were found. Proceeding without Argo Rollouts integration.",
			)
			serverCfg.RolloutsIntegrationEnabled = false
		} else {
			o.Logger.Debug("Argo Rollouts integration is enabled")
		}
	} else {
		o.Logger.Debug("Argo Rollouts integration is disabled")
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

	sender := natsevent.NewDefaultingEventSender(
		natsConn,
		event.EventsSubjectPrefix,
		"api",
	)
	defer sender.Shutdown()

	// The database holds Targets and ProjectConfigs. ProjectConfig writes
	// are announced over NATS.
	pool, err := database.Open(ctx, o.Database, "kargo-api")
	if err != nil {
		return err
	}
	defer pool.Close()
	configStore := projectconfigstore.New(pool, natsConn)

	// ProjectConfigs used to be Kubernetes resources. Copy any that exist
	// into the database, once, before anything can read or write the
	// database's. The import bypasses Kargo's RBAC: it acts for no user and
	// reads only ProjectConfig resources, Projects and one policy binding.
	if err = projectconfigs.NewImporter(
		pool,
		configStore,
		kubeClient.APIReader(),
		kubeClient.RESTMapper(),
	).Run(logging.ContextWithLogger(ctx, o.Logger)); err != nil {
		return fmt.Errorf("error importing ProjectConfig resources: %w", err)
	}

	srv := server.NewServer(
		serverCfg,
		kubeClient,
		rbac.NewKubernetesRolesDatabase(
			kubeClient,
			kubeClient.InternalClient(),
			rbac.RolesDatabaseConfigFromEnv(),
		),
		sender,
		database.New(pool),
		configStore,
		natsConn,
	)
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", o.BindAddress, o.Port))
	if err != nil {
		return fmt.Errorf("error creating listener: %w", err)
	}
	defer l.Close()

	if err = srv.Serve(ctx, l); err != nil {
		return fmt.Errorf("error serving API: %w", err)
	}
	return nil
}
