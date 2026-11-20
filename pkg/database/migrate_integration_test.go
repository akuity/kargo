//go:build integration && db

package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/db"
)

// freshDatabase creates an empty database on the server at DATABASE_URL and
// returns a connection string for it. The database is dropped afterward.
func freshDatabase(t *testing.T) string {
	t.Helper()
	adminURL := os.Getenv("DATABASE_URL")
	if adminURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	require.NoError(t, err)
	t.Cleanup(func() { admin.Close(ctx) })
	name := fmt.Sprintf("kargo_test_%d", time.Now().UnixNano())
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
	})
	// pgx's ConnConfig.ConnString returns the string it was parsed from, not
	// one reflecting later changes, so the URL is rewritten instead. Returning
	// the admin URL would run every test against the shared database.
	u, err := url.Parse(adminURL)
	require.NoError(t, err)
	u.Path = "/" + name
	return u.String()
}

func TestMigrator_integration(t *testing.T) {
	ctx := context.Background()
	connString := freshDatabase(t)
	pool, err := NewPool(ctx, connString, "kargo-test")
	require.NoError(t, err)
	defer pool.Close()
	migrations, err := db.Migrations()
	require.NoError(t, err)
	m, err := NewMigrator(ctx, pool, migrations, time.Minute)
	require.NoError(t, err)

	// A fresh database is behind.
	require.ErrorIs(t, m.Check(ctx), ErrSchemaOutOfDate)
	current, target, err := m.Version(ctx)
	require.NoError(t, err)
	require.Zero(t, current)
	require.Positive(t, target)

	// Applying brings it to the target and Check passes.
	applied, err := m.Up(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, applied)
	require.Equal(t, target, applied[len(applied)-1])
	require.NoError(t, m.Check(ctx))

	// Applying again is a no-op.
	applied, err = m.Up(ctx)
	require.NoError(t, err)
	require.Empty(t, applied)

	// The schema is usable through the generated queries.
	q := New(pool)
	require.NoError(t, q.UpsertProject(ctx, UpsertProjectParams{
		ID: "uid", Name: "example", CreatedAt: time.Now(),
	}))
	projects, err := q.ListProjects(ctx)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	require.Equal(t, "uid", projects[0].ID)
}

func TestMigrator_integration_concurrent(t *testing.T) {
	ctx := context.Background()
	connString := freshDatabase(t)
	migrations, err := db.Migrations()
	require.NoError(t, err)

	// Several runners start on the same empty database, as replicas or a Job
	// retry would. The session lock must serialize them so that exactly the
	// migrations are applied, once, and every runner exits cleanly.
	const runners = 4
	results := make([]error, runners)
	var wg sync.WaitGroup
	for i := range runners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool, err := NewPool(ctx, connString, fmt.Sprintf("runner-%d", i))
			if err != nil {
				results[i] = err
				return
			}
			defer pool.Close()
			m, err := NewMigrator(ctx, pool, migrations, time.Minute)
			if err != nil {
				results[i] = err
				return
			}
			_, results[i] = m.Up(ctx)
		}()
	}
	wg.Wait()
	for i, err := range results {
		require.NoError(t, err, "runner %d", i)
	}
	pool, err := NewPool(ctx, connString, "kargo-test")
	require.NoError(t, err)
	defer pool.Close()
	m, err := NewMigrator(ctx, pool, migrations, time.Minute)
	require.NoError(t, err)
	require.NoError(t, m.Check(ctx))
}

// withSearchPath returns connString with search_path pointing at schema,
// which is how instances sharing a database are told apart.
func withSearchPath(t *testing.T, connString, schema string) string {
	t.Helper()
	u, err := url.Parse(connString)
	require.NoError(t, err)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func TestMigrator_integration_schemaLock(t *testing.T) {
	ctx := context.Background()
	connString := freshDatabase(t)
	migrations, err := db.Migrations()
	require.NoError(t, err)

	// Two schemas in one database, as two instances sharing a server would
	// have.
	admin, err := pgx.Connect(ctx, connString)
	require.NoError(t, err)
	defer admin.Close(ctx)
	for _, schema := range []string{"tenant_a", "tenant_b"} {
		_, err = admin.Exec(ctx, "CREATE SCHEMA "+schema)
		require.NoError(t, err)
	}

	// Something holds tenant_a's migration lock for the duration of the test,
	// as a slow migration run against that schema would.
	holder, err := pgx.Connect(ctx, connString)
	require.NoError(t, err)
	defer holder.Close(ctx)
	_, err = holder.Exec(
		ctx, "SELECT pg_advisory_lock($1)", migrationLockID("tenant_a"),
	)
	require.NoError(t, err)

	// lockProbeInterval is the shortest permitted timeout and allows a single
	// attempt, so a blocked run fails fast instead of waiting on the holder.
	run := func(schema string) error {
		pool, runErr := NewPool(ctx, withSearchPath(t, connString, schema), schema)
		if runErr != nil {
			return runErr
		}
		defer pool.Close()
		m, runErr := NewMigrator(ctx, pool, migrations, lockProbeInterval)
		if runErr != nil {
			return runErr
		}
		_, runErr = m.Up(ctx)
		return runErr
	}

	// tenant_b is unaffected by tenant_a's lock.
	require.NoError(t, run("tenant_b"))
	// tenant_a waits on its own lock.
	require.ErrorContains(t, run("tenant_a"), "failed to acquire lock")

	// Releasing the lock lets tenant_a proceed, and each schema ends up with
	// its own copy of the schema.
	_, err = holder.Exec(
		ctx, "SELECT pg_advisory_unlock($1)", migrationLockID("tenant_a"),
	)
	require.NoError(t, err)
	require.NoError(t, run("tenant_a"))
	for _, schema := range []string{"tenant_a", "tenant_b"} {
		var exists bool
		require.NoError(t, admin.QueryRow(
			ctx,
			"SELECT to_regclass($1) IS NOT NULL",
			schema+".goose_db_version",
		).Scan(&exists))
		require.True(t, exists, schema)
	}
}
