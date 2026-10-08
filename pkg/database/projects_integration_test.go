//go:build integration && db

package database

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

func TestProjectsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	queries := New(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	project := UpsertProjectParams{ID: "opaque-project", Name: "demo", CreatedAt: created}
	// replace writes a Project the way the mirror does.
	replace := func(params UpsertProjectParams) error {
		if err := queries.DeleteReplacedProject(ctx, DeleteReplacedProjectParams{
			Name: params.Name,
			ID:   params.ID,
		}); err != nil {
			return err
		}
		return queries.UpsertProject(ctx, params)
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "idempotent writes preserve creation and refresh sync timestamps",
			run: func(t *testing.T) {
				require.NoError(t, replace(project))
				_, err := pool.Exec(ctx, "UPDATE projects SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				require.NoError(t, replace(project))
				ids, err := listProjectIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, ids)
				var creation, synced time.Time
				require.NoError(t, pool.QueryRow(ctx, "SELECT created_at, synced_at FROM projects").Scan(&creation, &synced))
				require.True(t, creation.Equal(created))
				require.True(t, synced.After(created))
			},
		},
		{
			name: "snapshot query includes all mirrored fields",
			run: func(t *testing.T) {
				require.NoError(t, replace(project))
				projects, err := queries.ListProjects(ctx)
				require.NoError(t, err)
				require.Len(t, projects, 1)
				require.Equal(t, project.ID, projects[0].ID)
				require.Equal(t, project.Name, projects[0].Name)
				require.True(t, projects[0].CreatedAt.Equal(created))
				require.False(t, projects[0].SyncedAt.IsZero())
			},
		},
		{
			name: "database enforces unique names and identities",
			run: func(t *testing.T) {
				require.NoError(t, replace(project))
				duplicate := project
				duplicate.ID = "another"
				requirePGError(t, queries.UpsertProject(ctx, duplicate), "23505")
				_, err := pool.Exec(ctx, "INSERT INTO projects SELECT * FROM projects")
				requirePGError(t, err, "23505")
			},
		},
		{
			// Deleting a Project deletes everything in its namespace, so rows
			// that belonged to the old Project must not survive its
			// recreation under the same name.
			name: "recreation replaces the row and cascades to the old Project's rows",
			run: func(t *testing.T) {
				_, err := pool.Exec(ctx, `CREATE TABLE project_children (
					project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE
				)`)
				require.NoError(t, err)
				require.NoError(t, replace(project))
				_, err = pool.Exec(ctx, "INSERT INTO project_children VALUES ($1)", project.ID)
				require.NoError(t, err)
				replacement := project
				replacement.ID = "new-project"
				require.NoError(t, replace(replacement))
				ids, err := listProjectIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{replacement.ID}, ids)
				var children int
				require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM project_children").Scan(&children))
				require.Zero(t, children)
			},
		},
		{
			name: "deletion by name",
			run: func(t *testing.T) {
				require.NoError(t, replace(project))
				other := UpsertProjectParams{ID: "other-project", Name: "other", CreatedAt: created}
				require.NoError(t, replace(other))
				require.NoError(t, queries.DeleteProjectByName(ctx, "other"))
				ids, err := listProjectIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, ids)
				require.NoError(t, queries.DeleteProjectByName(ctx, "absent"))
			},
		},
		{
			name: "blocked write respects the caller's deadline and recovers",
			run: func(t *testing.T) {
				require.NoError(t, replace(project))
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = tx.Exec(ctx, "SELECT id FROM projects FOR UPDATE")
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				require.ErrorIs(t, queries.UpsertProject(deadlineCtx, project), context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				require.NoError(t, replace(project))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "DROP TABLE IF EXISTS project_children")
			require.NoError(t, err)
			_, err = pool.Exec(ctx, "TRUNCATE projects CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func TestMigrationsIntegration(t *testing.T) {
	t.Parallel()
	pool, migrations := isolatedDatabase(t)
	ctx := context.Background()
	applied, err := migrations.Up(ctx) // Already-applied migrations are harmless.
	require.NoError(t, err)
	require.Empty(t, applied)
	_, err = migrations.Down(ctx)
	require.NoError(t, err)
	var table *string
	require.NoError(t, pool.QueryRow(ctx, "SELECT to_regclass('projects')::text").Scan(&table))
	require.Nil(t, table)
	_, err = migrations.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, New(pool).UpsertProject(ctx, UpsertProjectParams{
		ID: "after-rollback", Name: "demo", CreatedAt: time.Now(),
	}))
}

// isolatedDatabase creates a schema of its own for the test, applies the
// repository's migrations to it, and returns a pool scoped to that schema
// together with a migrator for it. The schema is dropped when the test ends.
func isolatedDatabase(t *testing.T) (*pgxpool.Pool, *goose.Provider) {
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
	pool, err := NewPool(ctx, isolatedDSN, "kargo-test")
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

func requirePGError(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, code, pgErr.Code)
}

func listProjectIDs(ctx context.Context, queries *Queries) ([]string, error) {
	rows, err := queries.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}
