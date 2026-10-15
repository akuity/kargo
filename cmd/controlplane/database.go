package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akuity/kargo/db"
	"github.com/akuity/kargo/pkg/database"
)

// openDatabase opens a pool to the configured database, waiting for it to
// accept connections and then for its schema to reach the version this
// binary expects, within one connect timeout for both. Nothing orders a
// component's start after the migration run, so the schema being behind is
// waited for like the database being down; refusing to serve until it has
// caught up is what keeps new code from ever running against tables the
// migrations have not created yet.
func openDatabase(
	ctx context.Context,
	cfg database.Config,
	applicationName string,
) (*pgxpool.Pool, error) {
	connString, err := cfg.ConnString()
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(cfg.ConnectTimeout)
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
	if err = database.WaitForSchema(ctx, migrator, time.Until(deadline)); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
