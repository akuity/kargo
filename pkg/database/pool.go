package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// applicationNameParam is the PostgreSQL session parameter that labels a
// connection in pg_stat_activity and in the server's logs.
const applicationNameParam = "application_name"

// NewPool opens a connection pool for the database at connString, which may
// be a URL or a key/value DSN, and confirms it can reach the database before
// returning. Every connection in the pool reports applicationName to the
// server so that its sessions can be told apart from other components' in
// pg_stat_activity and in lock-wait logs, and every operation issued through
// the pool records an OpenTelemetry span.
func NewPool(
	ctx context.Context,
	connString string,
	applicationName string,
) (*pgxpool.Pool, error) {
	cfg, err := newPoolConfig(connString, applicationName)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	return pool, nil
}

// newPoolConfig parses connString and attaches the tracer and application
// name. An application name already present in connString is left alone.
func newPoolConfig(
	connString string,
	applicationName string,
) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("error parsing database connection string: %w", err)
	}
	cfg.ConnConfig.Tracer = pgxTracer{}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	if _, ok := cfg.ConnConfig.RuntimeParams[applicationNameParam]; !ok {
		cfg.ConnConfig.RuntimeParams[applicationNameParam] = applicationName
	}
	return cfg, nil
}
