package main

import (
	"context"
	stdruntime "runtime"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/db"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/os"
	"github.com/akuity/kargo/pkg/telemetry"
	versionpkg "github.com/akuity/kargo/pkg/x/version"
)

type migrateOptions struct {
	Logger *logging.Logger
}

func newMigrateCommand() *cobra.Command {
	_, format := getLogVars()
	cmdOpts := &migrateOptions{
		// During startup, we enforce use of an info-level logger to ensure that
		// no important startup messages are missed.
		Logger: logging.NewLoggerOrDie(logging.InfoLevel, format),
	}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending database schema migrations and exit",
		Long: `Apply every database schema migration embedded in this binary that the
database has not yet seen, then exit. Concurrent invocations serialize on a
database-level lock, so running more than one at a time is safe.

The database is described by DATABASE_URL, or by DATABASE_HOST and its
companions. Connection attempts are retried until DATABASE_CONNECT_TIMEOUT
elapses, since the database may still be starting.`,
		DisableAutoGenTag: true,
		SilenceErrors:     true,
		SilenceUsage:      true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			version := versionpkg.GetVersion()

			cmdOpts.Logger.Info(
				"Starting Kargo database migration",
				"version", version.Version,
				"commit", version.GitCommit,
				"GOMAXPROCS", stdruntime.GOMAXPROCS(0),
				"GOMEMLIMIT", os.GetEnv("GOMEMLIMIT", ""),
			)

			cmdOpts.complete()

			return cmdOpts.run(cmd.Context())
		},
	}

	return cmd
}

func (o *migrateOptions) complete() {
	logLevel, logFormat := getLogVars()
	o.Logger = logging.NewLoggerOrDie(logLevel, logFormat)
}

func (o *migrateOptions) run(ctx context.Context) error {
	ctx = logging.ContextWithLogger(ctx, o.Logger)

	shutdownTelemetry, err := telemetry.SetupFromEnv(ctx, "migrate")
	if err != nil {
		return err
	}
	defer shutdownTelemetry()

	cfg := database.ConfigFromEnv()
	connString, err := cfg.ConnString()
	if err != nil {
		return err
	}
	pool, err := database.ConnectWithRetry(
		ctx,
		connString,
		"kargo-migrate",
		cfg.ConnectTimeout,
	)
	if err != nil {
		return err
	}
	defer pool.Close()

	migrations, err := db.Migrations()
	if err != nil {
		return err
	}
	migrator, err := database.NewMigrator(
		ctx,
		pool,
		migrations,
		cfg.MigrationLockTimeout,
	)
	if err != nil {
		return err
	}

	applied, err := migrator.Up(ctx)
	if err != nil {
		return err
	}
	current, _, err := migrator.Version(ctx)
	if err != nil {
		return err
	}
	o.Logger.Info(
		"Database schema is up to date",
		"applied", len(applied),
		"version", current,
	)
	return nil
}
