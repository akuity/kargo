package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/akuity/kargo/pkg/logging"
)

// ErrSchemaOutOfDate is returned by Migrator.Check when the database has not
// been migrated to the version this binary expects.
var ErrSchemaOutOfDate = errors.New("database schema is out of date")

// Migrator applies and inspects the database schema migrations embedded in
// this binary.
type Migrator interface {
	// Up applies every pending migration in order and returns the versions
	// it applied. It holds a session-level advisory lock for the duration, so
	// concurrent runners wait for one another instead of racing. It is safe to
	// call when nothing is pending.
	Up(ctx context.Context) ([]int64, error)
	// Version returns the version the database is at and the version this
	// binary's migrations reach.
	Version(ctx context.Context) (current int64, target int64, err error)
	// Check returns ErrSchemaOutOfDate if the database is behind the version
	// this binary expects. Components that read the database should call it at
	// startup and refuse to serve until it passes, so that new code never runs
	// against an old schema.
	Check(ctx context.Context) error
}

type migrator struct {
	provider *goose.Provider
}

// lockProbeInterval is how often a waiting migration run re-checks whether
// the database-level lock has been released.
const lockProbeInterval = 5 * time.Second

// NewMigrator returns a Migrator that applies the given migrations through
// the pool. Migrations run through pgx's database/sql adapter, so the pool's
// tracer sees them like any other query. lockTimeout bounds how long Up waits
// for a concurrent run to release the database-level lock; it must exceed the
// slowest migration, which is what a waiting run is waiting on.
func NewMigrator(
	ctx context.Context,
	pool *pgxpool.Pool,
	migrations fs.FS,
	lockTimeout time.Duration,
) (Migrator, error) {
	if lockTimeout < lockProbeInterval {
		return nil, fmt.Errorf(
			"migration lock timeout must be at least %s", lockProbeInterval,
		)
	}
	locker, err := lock.NewPostgresSessionLocker(
		lock.WithLockTimeout(
			uint64(lockProbeInterval.Seconds()),
			uint64(lockTimeout/lockProbeInterval),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating migration lock: %w", err)
	}
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		stdlib.OpenDBFromPool(pool),
		migrations,
		goose.WithSessionLocker(locker),
		goose.WithLogger(gooseLogger{
			logger: logging.LoggerFromContext(ctx),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing migrations: %w", err)
	}
	return &migrator{provider: provider}, nil
}

func (m *migrator) Up(ctx context.Context) ([]int64, error) {
	results, err := m.provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("error applying migrations: %w", err)
	}
	applied := make([]int64, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Version)
	}
	return applied, nil
}

func (m *migrator) Version(ctx context.Context) (int64, int64, error) {
	current, err := m.provider.GetDBVersion(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("error reading database schema version: %w", err)
	}
	return current, m.target(), nil
}

func (m *migrator) Check(ctx context.Context) error {
	current, target, err := m.Version(ctx)
	if err != nil {
		return err
	}
	if current < target {
		return fmt.Errorf(
			"%w: database is at version %d but this binary expects %d",
			ErrSchemaOutOfDate, current, target,
		)
	}
	return nil
}

// target returns the highest migration version this binary carries, or zero
// when it carries none.
func (m *migrator) target() int64 {
	var target int64
	for _, source := range m.provider.ListSources() {
		target = max(target, source.Version)
	}
	return target
}

// ConnectWithRetry opens a pool as NewPool does, but keeps retrying while the
// database is unreachable or still starting, until it accepts a connection or
// timeout elapses. Components start before the database is ready more often
// than not, so those failures are expected rather than fatal. Anything else,
// such as a malformed connection string, a wrong password, or a database that
// does not exist, will not fix itself and fails immediately.
func ConnectWithRetry(
	ctx context.Context,
	connString string,
	applicationName string,
	timeout time.Duration,
) (*pgxpool.Pool, error) {
	cfg, err := newPoolConfig(connString, applicationName)
	if err != nil {
		return nil, err
	}
	logger := logging.LoggerFromContext(ctx)
	deadline := time.Now().Add(timeout)
	for {
		pool, err := newPoolFromConfig(ctx, cfg)
		if err == nil {
			return pool, nil
		}
		if !isTransientConnectError(err) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf(
				"database did not become reachable within %s: %w", timeout, err,
			)
		}
		logger.Info("waiting for the database", "error", err.Error())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(connectRetryInterval):
		}
	}
}

// connectRetryInterval is how long ConnectWithRetry waits between attempts.
const connectRetryInterval = 2 * time.Second

// SQLSTATE codes the server answers with while it is not ready to take a
// connection yet, which clear on their own.
const (
	// sqlStateCannotConnectNow is "the database system is starting up" (or
	// shutting down).
	sqlStateCannotConnectNow = "57P03"
	// sqlStateTooManyConnections is the server's connection limit being hit,
	// which other components' retries would release.
	sqlStateTooManyConnections = "53300"
)

// isTransientConnectError reports whether a connection failure is worth
// waiting out. A failure to reach the server at all, such as a refused
// connection, a DNS miss, or a timeout, is transient: the database is most
// likely still starting. So is a server that answers but is not ready yet.
// Any other answer from the server, such as a rejected password or an unknown
// database, is a configuration error that no amount of waiting will fix.
func isTransientConnectError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return true
	}
	switch pgErr.Code {
	case sqlStateCannotConnectNow, sqlStateTooManyConnections:
		return true
	}
	return false
}

// gooseLogger adapts Kargo's logger to the interface Goose logs through.
type gooseLogger struct {
	logger *logging.Logger
}

func (l gooseLogger) Printf(format string, v ...any) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

func (l gooseLogger) Fatalf(format string, v ...any) {
	// Goose only calls this from its own CLI paths, never from the Provider
	// API this package uses, but the interface requires it.
	l.logger.Error(fmt.Errorf(format, v...), "fatal migration error")
}
