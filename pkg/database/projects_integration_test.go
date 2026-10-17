//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestProjectsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
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
				databasetest.RequirePGError(t, queries.UpsertProject(ctx, duplicate), "23505")
				_, err := pool.Exec(ctx, "INSERT INTO projects SELECT * FROM projects")
				databasetest.RequirePGError(t, err, "23505")
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
	pool, migrations := databasetest.IsolatedDatabase(t)
	ctx := context.Background()
	applied, err := migrations.Up(ctx) // Already-applied migrations are harmless.
	require.NoError(t, err)
	require.Empty(t, applied)
	// Every migration rolls back, one at a time and newest first, and the
	// schema then comes back up whole.
	for {
		version, versionErr := migrations.GetDBVersion(ctx)
		require.NoError(t, versionErr)
		if version == 0 {
			break
		}
		result, downErr := migrations.Down(ctx)
		require.NoError(t, downErr)
		require.Equal(t, version, result.Source.Version)
	}
	requireTable(t, pool, "projects", false)
	_, err = migrations.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, New(pool).UpsertProject(ctx, UpsertProjectParams{
		ID: "after-rollback", Name: "demo", CreatedAt: time.Now(),
	}))
}

// requireTable asserts whether a table exists in the test's schema.
func requireTable(t *testing.T, pool *pgxpool.Pool, name string, present bool) {
	t.Helper()
	var table *string
	require.NoError(t, pool.QueryRow(
		context.Background(), "SELECT to_regclass($1)::text", name,
	).Scan(&table))
	if present {
		require.NotNil(t, table, "table %q should exist", name)
	} else {
		require.Nil(t, table, "table %q should not exist", name)
	}
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
