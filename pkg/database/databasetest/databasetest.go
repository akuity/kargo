// Package databasetest helps integration tests run against PostgreSQL.
package databasetest

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/db"
)

// IsolatedDatabase creates a schema of its own for the test on the server at
// DATABASE_URL, applies the repository's migrations to it, and returns a pool
// scoped to that schema together with a migrator for it. The schema is
// dropped when the test ends. The test is skipped if DATABASE_URL is unset.
func IsolatedDatabase(t *testing.T) (*pgxpool.Pool, *goose.Provider) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(admin.Close)
	schema := "dbsync_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, cleanupErr := admin.Exec(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		require.NoError(t, cleanupErr)
	})
	uri, err := url.Parse(dsn)
	require.NoError(t, err)
	query := uri.Query()
	query.Set("search_path", schema)
	uri.RawQuery = query.Encode()
	isolatedDSN := uri.String()
	migrations := newMigrator(t, isolatedDSN)
	_, err = migrations.Up(ctx)
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, isolatedDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool, migrations
}

// newMigrator runs the embedded migrations against dsn in-process.
// Shelling out to `go tool goose` would compile the tool on first use, which
// on a cold CI runner takes longer than a test should wait.
func newMigrator(t *testing.T, dsn string) *goose.Provider {
	t.Helper()
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	migrations, err := db.Migrations()
	require.NoError(t, err)
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		stdlib.OpenDB(*cfg),
		migrations,
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, provider.Close()) })
	return provider
}

// RequirePGError asserts that err is a PostgreSQL error with the given
// SQLSTATE code.
func RequirePGError(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, code, pgErr.Code)
}
