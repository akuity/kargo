package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akuity/kargo/db"
	"github.com/akuity/kargo/pkg/database"
)

// openDatabase opens a pool to the configured database, waiting for it to
// accept connections, and confirms that its schema is at the version this
// binary expects. Refusing to start against an old schema is what keeps new
// code from ever running against tables the migrations have not created yet.
func openDatabase(
	ctx context.Context,
	cfg database.Config,
	applicationName string,
) (*pgxpool.Pool, error) {
	connString, err := cfg.ConnString()
	if err != nil {
		return nil, err
	}
	pool, err := database.ConnectWithRetry(
		ctx,
		connString,
		applicationName,
		cfg.ConnectTimeout,
	)
	if err != nil {
		return nil, err
	}
	migrations, err := db.Migrations()
	if err != nil {
		pool.Close()
		return nil, err
	}
	migrator, err := database.NewMigrator(
		ctx,
		pool,
		migrations,
		cfg.MigrationLockTimeout,
	)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err = migrator.Check(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
