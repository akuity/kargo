package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akuity/kargo/db"
)

// Open opens a pool to the configured database for a component that serves
// from it, waiting for the database to accept connections and then for its
// schema to reach the version this binary's embedded migrations expect,
// within one ConnectTimeout for both. Nothing orders a component's start
// after the migration run, so a schema that is behind is waited for like a
// database that is down; refusing to serve until it has caught up keeps new
// code from ever running against tables the migrations have not created yet.
// A schema still behind at the deadline is ErrSchemaOutOfDate.
//
// The caller closes the pool. The migrate command, which brings the schema
// up rather than waiting for it, uses ConnectWithRetry and NewMigrator
// directly instead.
func Open(ctx context.Context, cfg Config, applicationName string) (*pgxpool.Pool, error) {
	connString, err := cfg.ConnString()
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(cfg.ConnectTimeout)
	pool, err := ConnectWithRetry(ctx, connString, applicationName, cfg.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	migrations, err := db.Migrations()
	if err != nil {
		pool.Close()
		return nil, err
	}
	migrator, err := NewMigrator(ctx, pool, migrations, cfg.MigrationLockTimeout)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err = WaitForSchema(ctx, migrator, time.Until(deadline)); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
