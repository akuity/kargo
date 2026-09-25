//go:build integration

package database

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	store := NewStore(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	project := UpsertProjectParams{ID: "opaque-project", Name: "demo", CreatedAt: created}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "idempotent writes preserve creation and refresh sync timestamps",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				_, err := pool.Exec(ctx, "UPDATE projects SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				require.NoError(t, store.UpsertProject(ctx, project))
				ids, err := listProjectIDs(ctx, store)
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
				require.NoError(t, store.UpsertProject(ctx, project))
				projects, err := store.ListProjects(ctx)
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
				require.NoError(t, store.UpsertProject(ctx, project))
				duplicate := project
				duplicate.ID = "another"
				requirePGError(t, New(pool).UpsertProject(ctx, duplicate), "23505")
				_, err := pool.Exec(ctx, "INSERT INTO projects SELECT * FROM projects")
				requirePGError(t, err, "23505")
			},
		},
		{
			name: "deletion by name",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				other := UpsertProjectParams{ID: "other-project", Name: "other", CreatedAt: created}
				require.NoError(t, store.UpsertProject(ctx, other))
				require.NoError(t, store.DeleteProjectByName(ctx, "other"))
				ids, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, ids)
				require.NoError(t, store.DeleteProjectByName(ctx, "absent"))
			},
		},
		{
			name: "recreation replaces identities",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				replacement := project
				replacement.ID = "new-project"
				require.NoError(t, store.UpsertProject(ctx, replacement))
				ids, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{replacement.ID}, ids)
			},
		},
		{
			name: "failed replacement rolls back deletion",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				_, err := pool.Exec(ctx, "ALTER TABLE projects ADD CONSTRAINT reject_new CHECK (id <> 'rejected')")
				require.NoError(t, err)
				rejected := project
				rejected.ID = "rejected"
				requirePGError(t, store.UpsertProject(ctx, rejected), "23514")
				ids, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, ids)
			},
		},
		{
			name: "cleanup deletes only candidate IDs",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.DeleteProjectsByID(ctx, []string{"absent"}))
				ids, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, ids)
				require.NoError(t, store.DeleteProjectsByID(ctx, []string{project.ID}))
				ids, err = listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Empty(t, ids)
			},
		},
		{
			name: "blocked transaction respects caller deadline and recovers",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = tx.Exec(ctx, "SELECT id FROM projects FOR UPDATE")
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				require.ErrorIs(t, store.UpsertProject(deadlineCtx, project), context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				require.NoError(t, store.UpsertProject(ctx, project))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE projects CASCADE")
			require.NoError(t, err)
			_, err = pool.Exec(ctx, "ALTER TABLE projects DROP CONSTRAINT IF EXISTS reject_new")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func TestMigrationsIntegration(t *testing.T) {
	t.Parallel()
	pool, dsn := isolatedDatabase(t)
	runGoose(t, dsn, "up") // Already-applied migrations are harmless.
	runGoose(t, dsn, "down")
	var table *string
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT to_regclass('projects')::text").Scan(&table))
	require.Nil(t, table)
	runGoose(t, dsn, "up")
	require.NoError(t, NewStore(pool).UpsertProject(context.Background(), UpsertProjectParams{
		ID: "after-rollback", Name: "demo", CreatedAt: time.Now(),
	}))
}

func isolatedDatabase(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, dsn, "set TEST_DATABASE_URL to run database integration tests")
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(admin.Close)
	schema := "dbsync_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), operationTimeout)
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
	runGoose(t, isolatedDSN, "up")
	pool, err := NewPool(ctx, isolatedDSN, "kargo-test")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool, isolatedDSN
}

func runGoose(t *testing.T, dsn, command string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "tool", "goose", "-dir", "../../db/migrations", command)
	cmd.Env = append(os.Environ(), "GOOSE_DRIVER=postgres", "GOOSE_DBSTRING="+dsn)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func requirePGError(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, code, pgErr.Code)
}

func listProjectIDs(ctx context.Context, store Store) ([]string, error) {
	rows, err := store.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}
