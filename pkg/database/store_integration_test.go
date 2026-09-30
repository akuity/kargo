//go:build integration

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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/db"
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
	pool, migrations := isolatedDatabase(t)
	ctx := context.Background()
	applied, err := migrations.Up(ctx) // Already-applied migrations are harmless.
	require.NoError(t, err)
	require.Empty(t, applied)
	// Migrations roll back one at a time, newest first, and all the way down.
	_, err = migrations.Down(ctx)
	require.NoError(t, err)
	requireTable(t, pool, "targets", false)
	requireTable(t, pool, "projects", true)
	_, err = migrations.DownTo(ctx, 0)
	require.NoError(t, err)
	requireTable(t, pool, "projects", false)
	_, err = migrations.Up(ctx)
	require.NoError(t, err)
	store := NewStore(pool)
	require.NoError(t, store.UpsertProject(ctx, UpsertProjectParams{
		ID: "after-rollback", Name: "demo", CreatedAt: time.Now(),
	}))
	_, err = store.CreateTarget(ctx, "demo", &kargoapi.Target{
		ObjectMeta: metav1.ObjectMeta{Name: "us-east"},
	})
	require.NoError(t, err)
}

func TestTargetStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	store := NewStore(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	project := UpsertProjectParams{ID: "opaque-project", Name: "demo", CreatedAt: created}
	other := UpsertProjectParams{ID: "other-project", Name: "other", CreatedAt: created}
	newTarget := func(name string) *kargoapi.Target {
		return &kargoapi.Target{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"region": "us"},
			},
			Spec: kargoapi.TargetSpec{
				Params: map[string]apiextensionsv1.JSON{
					"cluster": {Raw: []byte(`{"replicas": 3, "name": "` + name + `"}`)},
				},
			},
		}
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create returns the stored Target",
			run: func(t *testing.T) {
				target, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.Equal(t, "demo", target.Namespace)
				require.Equal(t, "us-east", target.Name)
				_, err = uuid.Parse(string(target.UID))
				require.NoError(t, err)
				require.NotEmpty(t, target.ResourceVersion)
				require.False(t, target.CreationTimestamp.IsZero())
				require.Equal(t, map[string]string{"region": "us"}, target.Labels)
				require.JSONEq(t,
					`{"replicas": 3, "name": "us-east"}`,
					string(target.Spec.Params["cluster"].Raw),
				)
				var createdAt, updatedAt time.Time
				require.NoError(t, pool.QueryRow(
					ctx, "SELECT created_at, updated_at FROM targets",
				).Scan(&createdAt, &updatedAt))
				require.True(t, createdAt.Equal(updatedAt))
			},
		},
		{
			name: "empty maps round trip as absent",
			run: func(t *testing.T) {
				target, err := store.CreateTarget(ctx, "demo", &kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{Name: "bare"},
				})
				require.NoError(t, err)
				require.Nil(t, target.Labels)
				require.Nil(t, target.Spec.Params)
				fetched, err := store.GetTarget(ctx, "demo", "bare")
				require.NoError(t, err)
				require.Equal(t, target, fetched)
			},
		},
		{
			name: "a Project that is not mirrored yet",
			run: func(t *testing.T) {
				_, err := store.CreateTarget(ctx, "missing", newTarget("us-east"))
				require.ErrorIs(t, err, ErrProjectNotMirrored)
			},
		},
		{
			name: "names are unique within a Project only",
			run: func(t *testing.T) {
				_, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.ErrorIs(t, err, ErrAlreadyExists)
				_, err = store.CreateTarget(ctx, "other", newTarget("us-east"))
				require.NoError(t, err)
			},
		},
		{
			name: "the database rejects labels that are not a string map",
			run: func(t *testing.T) {
				for _, bad := range []string{`[]`, `{"a": 1}`, `{"a": null}`} {
					_, err := pool.Exec(ctx,
						"INSERT INTO targets (project_id, name, labels) VALUES ($1, 'x', $2)",
						project.ID, bad,
					)
					requirePGError(t, err, "23514")
				}
				_, err := pool.Exec(ctx,
					"INSERT INTO targets (project_id, name, params) VALUES ($1, 'x', '[]')",
					project.ID,
				)
				requirePGError(t, err, "23514")
			},
		},
		{
			name: "a value the database cannot store is invalid",
			run: func(t *testing.T) {
				target := newTarget("nul")
				target.Spec.Params["cluster"] = apiextensionsv1.JSON{Raw: []byte(`"a\u0000b"`)}
				_, err := store.CreateTarget(ctx, "demo", target)
				require.ErrorIs(t, err, ErrInvalid)
			},
		},
		{
			name: "list is per Project and ordered by name",
			run: func(t *testing.T) {
				for _, name := range []string{"b", "c", "a"} {
					_, err := store.CreateTarget(ctx, "demo", newTarget(name))
					require.NoError(t, err)
				}
				_, err := store.CreateTarget(ctx, "other", newTarget("z"))
				require.NoError(t, err)
				targets, err := store.ListTargets(ctx, "demo")
				require.NoError(t, err)
				require.Equal(t, []string{"a", "b", "c"}, targetNames(targets))
				targets, err = store.ListTargets(ctx, "missing")
				require.NoError(t, err)
				require.Empty(t, targets)
			},
		},
		{
			name: "get reports a missing Target",
			run: func(t *testing.T) {
				_, err := store.GetTarget(ctx, "demo", "absent")
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name: "updates keep identity and advance the resource version",
			run: func(t *testing.T) {
				created, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				first, err := store.UpdateTarget(ctx, "demo", &kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "us-east",
						Labels: map[string]string{"region": "eu"},
					},
				})
				require.NoError(t, err)
				second, err := store.UpdateTarget(ctx, "demo", &kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{Name: "us-east"},
				})
				require.NoError(t, err)
				require.Equal(t, created.UID, second.UID)
				require.Equal(t, created.CreationTimestamp, second.CreationTimestamp)
				require.Equal(t, map[string]string{"region": "eu"}, first.Labels)
				require.Nil(t, second.Labels)
				require.Nil(t, second.Spec.Params)
				require.Less(t, created.ResourceVersion, first.ResourceVersion)
				require.Less(t, first.ResourceVersion, second.ResourceVersion)
			},
		},
		{
			name: "the resource version advances even when the clock does not",
			run: func(t *testing.T) {
				created, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "UPDATE targets SET updated_at = '2999-01-01'")
				require.NoError(t, err)
				future, err := store.GetTarget(ctx, "demo", "us-east")
				require.NoError(t, err)
				updated, err := store.UpdateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.Less(t, created.ResourceVersion, future.ResourceVersion)
				require.Less(t, future.ResourceVersion, updated.ResourceVersion)
			},
		},
		{
			name: "update preconditions",
			run: func(t *testing.T) {
				created, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				stale := newTarget("us-east")
				stale.ResourceVersion = "1"
				_, err = store.UpdateTarget(ctx, "demo", stale)
				require.ErrorIs(t, err, ErrConflict)
				recreated := newTarget("us-east")
				recreated.UID = "00000000-0000-0000-0000-000000000000"
				_, err = store.UpdateTarget(ctx, "demo", recreated)
				require.ErrorIs(t, err, ErrConflict)
				current := newTarget("us-east")
				current.UID = created.UID
				current.ResourceVersion = created.ResourceVersion
				_, err = store.UpdateTarget(ctx, "demo", current)
				require.NoError(t, err)
				_, err = store.UpdateTarget(ctx, "demo", newTarget("absent"))
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name: "delete",
			run: func(t *testing.T) {
				_, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.NoError(t, store.DeleteTarget(ctx, "demo", "us-east"))
				require.ErrorIs(t, store.DeleteTarget(ctx, "demo", "us-east"), ErrNotFound)
				_, err = store.GetTarget(ctx, "demo", "us-east")
				require.ErrorIs(t, err, ErrNotFound)
			},
		},
		{
			name: "Targets go with their Project",
			run: func(t *testing.T) {
				_, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = store.CreateTarget(ctx, "other", newTarget("us-east"))
				require.NoError(t, err)
				require.NoError(t, store.DeleteProjectByName(ctx, "other"))
				targets, err := store.ListTargets(ctx, "other")
				require.NoError(t, err)
				require.Empty(t, targets)
				// A Project recreated under the same name is a new Project.
				replacement := project
				replacement.ID = "new-project"
				require.NoError(t, store.UpsertProject(ctx, replacement))
				targets, err = store.ListTargets(ctx, "demo")
				require.NoError(t, err)
				require.Empty(t, targets)
			},
		},
		{
			name: "a locked row respects the caller's deadline",
			run: func(t *testing.T) {
				_, err := store.CreateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = tx.Exec(ctx, "SELECT id FROM targets FOR UPDATE")
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				_, err = store.UpdateTarget(deadlineCtx, "demo", newTarget("us-east"))
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				_, err = store.UpdateTarget(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE projects CASCADE")
			require.NoError(t, err)
			require.NoError(t, store.UpsertProject(ctx, project))
			require.NoError(t, store.UpsertProject(ctx, other))
			testCase.run(t)
		})
	}
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

func targetNames(targets []kargoapi.Target) []string {
	names := make([]string, len(targets))
	for i, target := range targets {
		names[i] = target.Name
	}
	return names
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
