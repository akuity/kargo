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
	stage := UpsertStageParams{ID: "opaque-stage", ProjectID: project.ID, Name: "dev", CreatedAt: created}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "idempotent writes preserve creation and refresh sync timestamps",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				_, err := pool.Exec(ctx, "UPDATE projects SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "UPDATE stages SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				projectIDs, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, projectIDs)
				stageIDs, err := listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{stage.ID}, stageIDs)
				for _, query := range []string{
					"SELECT created_at, synced_at FROM projects",
					"SELECT created_at, synced_at FROM stages",
				} {
					var creation, synced time.Time
					require.NoError(t, pool.QueryRow(ctx, query).Scan(&creation, &synced))
					require.True(t, creation.Equal(created))
					require.True(t, synced.After(created))
				}
			},
		},
		{
			name: "snapshot queries include all mirrored fields",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				projects, err := store.ListProjects(ctx)
				require.NoError(t, err)
				require.Len(t, projects, 1)
				require.Equal(t, project.ID, projects[0].ID)
				require.Equal(t, project.Name, projects[0].Name)
				require.True(t, projects[0].CreatedAt.Equal(created))
				require.False(t, projects[0].SyncedAt.IsZero())
				stages, err := store.ListStages(ctx)
				require.NoError(t, err)
				require.Len(t, stages, 1)
				require.Equal(t, stage.ID, stages[0].ID)
				require.Equal(t, stage.Name, stages[0].Name)
				require.Equal(t, project.ID, stages[0].ProjectID)
				require.True(t, stages[0].CreatedAt.Equal(created))
				require.False(t, stages[0].SyncedAt.IsZero())
			},
		},
		{
			name: "stage arriving before parent can retry",
			run: func(t *testing.T) {
				requirePGError(t, store.UpsertStage(ctx, stage), "23503")
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
			},
		},
		{
			name: "database enforces unique names and identities",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				queries := New(pool)
				duplicateProject := project
				duplicateProject.ID = "another"
				requirePGError(t, queries.UpsertProject(ctx, duplicateProject), "23505")
				duplicateStage := stage
				duplicateStage.ID = "another"
				requirePGError(t, queries.UpsertStage(ctx, duplicateStage), "23505")
				_, err := pool.Exec(ctx, "INSERT INTO projects SELECT * FROM projects")
				requirePGError(t, err, "23505")
			},
		},
		{
			name: "same stage name in different projects and scoped deletion",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				otherProject := UpsertProjectParams{ID: "other-project", Name: "other", CreatedAt: created}
				otherStage := UpsertStageParams{ID: "other-stage", ProjectID: otherProject.ID, Name: "dev", CreatedAt: created}
				require.NoError(t, store.UpsertProject(ctx, otherProject))
				require.NoError(t, store.UpsertStage(ctx, otherStage))
				require.NoError(t, store.DeleteStageByName(ctx, DeleteStageByNameParams{ProjectName: "demo", Name: "dev"}))
				ids, err := listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{otherStage.ID}, ids)
				require.NoError(t, store.DeleteProjectByName(ctx, "other"))
				ids, err = listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Empty(t, ids)
			},
		},
		{
			name: "recreation replaces identities and cascades old children",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				replacementStage := stage
				replacementStage.ID = "new-stage"
				require.NoError(t, store.UpsertStage(ctx, replacementStage))
				stageIDs, err := listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{replacementStage.ID}, stageIDs)
				replacementProject := project
				replacementProject.ID = "new-project"
				require.NoError(t, store.UpsertProject(ctx, replacementProject))
				projectIDs, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{replacementProject.ID}, projectIDs)
				stageIDs, err = listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Empty(t, stageIDs)
			},
		},
		{
			name: "failed replacement rolls back deletion and cascading deletion",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				_, err := pool.Exec(ctx, "ALTER TABLE projects ADD CONSTRAINT reject_new CHECK (id <> 'rejected')")
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "ALTER TABLE stages ADD CONSTRAINT reject_new CHECK (id <> 'rejected')")
				require.NoError(t, err)
				rejectedProject := project
				rejectedProject.ID = "rejected"
				requirePGError(t, store.UpsertProject(ctx, rejectedProject), "23514")
				rejectedStage := stage
				rejectedStage.ID = "rejected"
				requirePGError(t, store.UpsertStage(ctx, rejectedStage), "23514")
				projectIDs, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{project.ID}, projectIDs)
				stageIDs, err := listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{stage.ID}, stageIDs)
			},
		},
		{
			name: "cleanup deletes only candidate IDs",
			run: func(t *testing.T) {
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertStage(ctx, stage))
				require.NoError(t, store.DeleteProjectsByID(ctx, []string{"absent"}))
				require.NoError(t, store.DeleteStagesByID(ctx, []string{"absent"}))
				stageIDs, err := listStageIDs(ctx, store)
				require.NoError(t, err)
				require.Equal(t, []string{stage.ID}, stageIDs)
				require.NoError(t, store.DeleteStagesByID(ctx, []string{stage.ID}))
				require.NoError(t, store.DeleteProjectsByID(ctx, []string{project.ID}))
				projectIDs, err := listProjectIDs(ctx, store)
				require.NoError(t, err)
				require.Empty(t, projectIDs)
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
			testCase.run(t)
		})
	}
}

func TestMigrationsIntegration(t *testing.T) {
	t.Parallel()
	pool, dsn := isolatedDatabase(t)
	runGoose(t, dsn, "up") // Already-applied migrations are harmless.
	var table *string
	requireTables := func(present bool, names ...string) {
		t.Helper()
		for _, name := range names {
			require.NoError(t, pool.QueryRow(context.Background(), "SELECT to_regclass($1)::text", name).Scan(&table))
			if present {
				require.NotNil(t, table, name)
			} else {
				require.Nil(t, table, name)
			}
		}
	}
	runGoose(t, dsn, "down")
	requireTables(false, "targets", "promotion_requests", "promotion_request_targets")
	requireTables(true, "warehouses", "freight", "projects")
	runGoose(t, dsn, "down")
	requireTables(
		false,
		"warehouses", "freight", "freight_commits", "freight_images", "freight_charts", "freight_artifacts",
	)
	requireTables(true, "projects")
	runGoose(t, dsn, "down")
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
	pool, err := NewPool(ctx, isolatedDSN)
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

func listStageIDs(ctx context.Context, store Store) ([]string, error) {
	rows, err := store.ListStages(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}
