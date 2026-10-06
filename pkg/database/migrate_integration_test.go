//go:build integration && db

package database

import (
	"context"
	"fmt"
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
	cfg, err := pgx.ParseConfig(adminURL)
	require.NoError(t, err)
	cfg.Database = name
	return cfg.ConnString()
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
	project, err := q.GetProjectByName(ctx, "example")
	require.NoError(t, err)
	require.Equal(t, "uid", project.ID)
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
