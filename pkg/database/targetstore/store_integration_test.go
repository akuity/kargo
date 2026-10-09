//go:build integration && db

package targetstore

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	store := New(queries)
	ctx := context.Background()
	newTarget := func(name string) database.TargetRow {
		return database.TargetRow{
			Name:   name,
			Labels: map[string]string{"region": "us"},
			Params: json.RawMessage(`{"cluster": {"replicas": 3, "name": "` + name + `"}}`),
		}
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create returns the stored Target",
			run: func(t *testing.T) {
				target, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.Equal(t, "demo", target.ProjectName)
				require.Equal(t, "us-east", target.Name)
				require.NotEqual(t, uuid.Nil, target.ID)
				require.False(t, target.CreatedAt.IsZero())
				require.True(t, target.CreatedAt.Equal(target.UpdatedAt))
				require.Equal(t, map[string]string{"region": "us"}, target.Labels)
				require.JSONEq(t,
					`{"cluster": {"replicas": 3, "name": "us-east"}}`,
					string(target.Params),
				)
			},
		},
		{
			name: "absent labels and params are stored as empty objects",
			run: func(t *testing.T) {
				target, err := store.Create(ctx, "demo", database.TargetRow{Name: "bare"})
				require.NoError(t, err)
				require.Empty(t, target.Labels)
				require.JSONEq(t, `{}`, string(target.Params))
				fetched, err := store.Get(ctx, "demo", "bare")
				require.NoError(t, err)
				require.Equal(t, target, fetched)
			},
		},
		{
			name: "names are unique within a Project only",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = store.Create(ctx, "demo", newTarget("us-east"))
				require.ErrorIs(t, err, database.ErrAlreadyExists)
				_, err = store.Create(ctx, "other", newTarget("us-east"))
				require.NoError(t, err)
			},
		},
		{
			name: "the database rejects labels that are not a string map",
			run: func(t *testing.T) {
				for _, bad := range []string{`[]`, `{"a": 1}`, `{"a": null}`} {
					_, err := pool.Exec(ctx,
						"INSERT INTO targets (project_name, name, labels) VALUES ('demo', 'x', $1)",
						bad,
					)
					databasetest.RequirePGError(t, err, "23514")
				}
				_, err := pool.Exec(ctx,
					"INSERT INTO targets (project_name, name, params) VALUES ('demo', 'x', '[]')",
				)
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "a value the database cannot store is invalid",
			run: func(t *testing.T) {
				target := newTarget("nul")
				target.Params = json.RawMessage(`{"cluster": "a\u0000b"}`)
				_, err := store.Create(ctx, "demo", target)
				require.ErrorIs(t, err, database.ErrInvalid)
			},
		},
		{
			name: "list is per Project and ordered by name",
			run: func(t *testing.T) {
				for _, name := range []string{"b", "c", "a"} {
					_, err := store.Create(ctx, "demo", newTarget(name))
					require.NoError(t, err)
				}
				_, err := store.Create(ctx, "other", newTarget("z"))
				require.NoError(t, err)
				targets, err := store.List(ctx, "demo")
				require.NoError(t, err)
				require.Equal(t, []string{"a", "b", "c"}, targetNames(targets))
				targets, err = store.List(ctx, "missing")
				require.NoError(t, err)
				require.Empty(t, targets)
			},
		},
		{
			name: "get reports a missing Target",
			run: func(t *testing.T) {
				_, err := store.Get(ctx, "demo", "absent")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "updates keep identity and advance updated_at",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				first, err := store.Update(ctx, "demo", database.TargetRow{
					Name:   "us-east",
					Labels: map[string]string{"region": "eu"},
				})
				require.NoError(t, err)
				second, err := store.Update(ctx, "demo", database.TargetRow{Name: "us-east"})
				require.NoError(t, err)
				require.Equal(t, created.ID, second.ID)
				require.True(t, created.CreatedAt.Equal(second.CreatedAt))
				require.Equal(t, map[string]string{"region": "eu"}, first.Labels)
				require.Empty(t, second.Labels)
				require.JSONEq(t, `{}`, string(second.Params))
				require.True(t, first.UpdatedAt.After(created.UpdatedAt))
				require.True(t, second.UpdatedAt.After(first.UpdatedAt))
			},
		},
		{
			name: "updated_at advances even when the clock does not",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "UPDATE targets SET updated_at = '2999-01-01'")
				require.NoError(t, err)
				future, err := store.Get(ctx, "demo", "us-east")
				require.NoError(t, err)
				updated, err := store.Update(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.True(t, future.UpdatedAt.After(created.UpdatedAt))
				require.True(t, updated.UpdatedAt.After(future.UpdatedAt))
			},
		},
		{
			name: "update preconditions",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				stale := newTarget("us-east")
				stale.UpdatedAt = created.UpdatedAt.Add(-time.Microsecond)
				_, err = store.Update(ctx, "demo", stale)
				require.ErrorIs(t, err, database.ErrConflict)
				recreated := newTarget("us-east")
				recreated.ID = uuid.New()
				_, err = store.Update(ctx, "demo", recreated)
				require.ErrorIs(t, err, database.ErrConflict)
				current := newTarget("us-east")
				current.ID = created.ID
				current.UpdatedAt = created.UpdatedAt
				_, err = store.Update(ctx, "demo", current)
				require.NoError(t, err)
				_, err = store.Update(ctx, "demo", newTarget("absent"))
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "delete",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.NoError(t, store.Delete(ctx, "demo", "us-east"))
				require.ErrorIs(t, store.Delete(ctx, "demo", "us-east"), database.ErrNotFound)
				_, err = store.Get(ctx, "demo", "us-east")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "calls join the caller's transaction",
			run: func(t *testing.T) {
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = New(queries.WithTx(tx)).Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.NoError(t, tx.Rollback(ctx))
				_, err = store.Get(ctx, "demo", "us-east")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "a locked row respects the caller's deadline",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = tx.Exec(ctx, "SELECT id FROM targets FOR UPDATE")
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				_, err = store.Update(deadlineCtx, "demo", newTarget("us-east"))
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				_, err = store.Update(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE targets")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func targetNames(targets []database.TargetRow) []string {
	names := make([]string, len(targets))
	for i, target := range targets {
		names[i] = target.Name
	}
	return names
}
