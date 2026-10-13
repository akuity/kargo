//go:build integration && db

package promotionrolloutstore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
	"github.com/akuity/kargo/pkg/database/targetstore"
)

func TestStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	store := New(queries)
	targets := targetstore.New(queries)
	ctx := context.Background()

	// seedTargets creates Targets in the Project and returns their ids by
	// name.
	seedTargets := func(t *testing.T, project string, names ...string) map[string]uuid.UUID {
		t.Helper()
		ids := map[string]uuid.UUID{}
		for _, name := range names {
			target, err := targets.Create(ctx, project, database.TargetRow{Name: name})
			require.NoError(t, err)
			ids[name] = target.ID
		}
		return ids
	}
	newRollout := func(stage, name string) database.PromotionRolloutRow {
		return database.PromotionRolloutRow{
			Name:          name,
			StageName:     stage,
			FreightName:   "freight-1",
			MaxConcurrent: 2,
			CreatedBy:     "tester",
		}
	}

	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create returns the stored rollout, unresolved, and resolve records its Targets in name order",
			run: func(t *testing.T) {
				ids := seedTargets(t, "create", "us-west", "us-east")
				rollout, err := store.Create(ctx, "create", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				require.Nil(t, rollout.TargetsResolvedAt)
				require.NotNil(t, rollout.Targets)
				require.Empty(t, rollout.Targets)
				unresolved := rollout
				rollout, err = store.ResolveTargets(
					ctx, rollout, []uuid.UUID{ids["us-west"], ids["us-east"], ids["us-west"]},
				)
				require.NoError(t, err)
				require.NotNil(t, rollout.TargetsResolvedAt)
				require.True(t, rollout.UpdatedAt.After(unresolved.UpdatedAt))
				require.Equal(t, "create", rollout.ProjectName)
				require.Equal(t, "prod.01A.abc", rollout.Name)
				require.Equal(t, "prod", rollout.StageName)
				require.EqualValues(t, 1, rollout.Number)
				require.Equal(t, "freight-1", rollout.FreightName)
				require.EqualValues(t, 2, rollout.MaxConcurrent)
				require.Equal(t, "tester", rollout.CreatedBy)
				require.NotEqual(t, uuid.Nil, rollout.ID)
				require.False(t, rollout.CreatedAt.IsZero())
				require.True(t, unresolved.CreatedAt.Equal(unresolved.UpdatedAt))
				require.Equal(t, "Pending", rollout.StatusPhase)
				require.Nil(t, rollout.StatusStartedAt)
				require.Nil(t, rollout.StatusFinishedAt)
				require.Equal(t, []string{"us-east", "us-west"}, targetNames(rollout))
				require.Equal(t, ids["us-east"], rollout.Targets[0].TargetID)

				fetched, err := store.Get(ctx, "create", "prod.01A.abc")
				require.NoError(t, err)
				require.Equal(t, rollout, fetched)
			},
		},
		{
			name: "numbers count per Stage and survive a failed create",
			run: func(t *testing.T) {
				first, err := store.Create(ctx, "numbers", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				require.EqualValues(t, 1, first.Number)
				second, err := store.Create(ctx, "numbers", newRollout("prod", "prod.01B.abc"))
				require.NoError(t, err)
				require.EqualValues(t, 2, second.Number)
				other, err := store.Create(ctx, "numbers", newRollout("staging", "staging.01A.abc"))
				require.NoError(t, err)
				require.EqualValues(t, 1, other.Number)

				// A duplicate name fails the whole statement, increment included.
				_, err = store.Create(ctx, "numbers", newRollout("prod", "prod.01B.abc"))
				require.ErrorIs(t, err, database.ErrAlreadyExists)
				third, err := store.Create(ctx, "numbers", newRollout("prod", "prod.01C.abc"))
				require.NoError(t, err)
				require.EqualValues(t, 3, third.Number)
			},
		},
		{
			name: "concurrent creates for one Stage are numbered without gaps",
			run: func(t *testing.T) {
				const n = 8
				var wg sync.WaitGroup
				numbers := make([]int64, n)
				errs := make([]error, n)
				for i := range n {
					wg.Go(func() {
						rollout, err := store.Create(
							ctx, "concurrent", newRollout("prod", "prod."+uuid.NewString()),
						)
						numbers[i], errs[i] = rollout.Number, err
					})
				}
				wg.Wait()
				seen := map[int64]bool{}
				for i := range n {
					require.NoError(t, errs[i])
					seen[numbers[i]] = true
				}
				for i := int64(1); i <= n; i++ {
					require.True(t, seen[i], "number %d was not assigned", i)
				}
			},
		},
		{
			name: "resolving to no Targets marks the rollout resolved with an empty list",
			run: func(t *testing.T) {
				rollout, err := store.Create(ctx, "empty", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				rollout, err = store.ResolveTargets(ctx, rollout, nil)
				require.NoError(t, err)
				require.NotNil(t, rollout.TargetsResolvedAt)
				require.NotNil(t, rollout.Targets)
				require.Empty(t, rollout.Targets)
			},
		},
		{
			name: "resolve writes exactly once",
			run: func(t *testing.T) {
				ids := seedTargets(t, "once", "us-east", "us-west")
				rollout, err := store.Create(ctx, "once", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				resolved, err := store.ResolveTargets(ctx, rollout, []uuid.UUID{ids["us-east"]})
				require.NoError(t, err)
				_, err = store.ResolveTargets(ctx, rollout, []uuid.UUID{ids["us-west"]})
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "already resolved")
				fetched, err := store.Get(ctx, "once", "prod.01A.abc")
				require.NoError(t, err)
				require.Equal(t, resolved, fetched)
				require.Equal(t, []string{"us-east"}, targetNames(fetched))
			},
		},
		{
			name: "names are unique within a Project only",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "unique-a", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				_, err = store.Create(ctx, "unique-a", newRollout("prod", "prod.01A.abc"))
				require.ErrorIs(t, err, database.ErrAlreadyExists)
				_, err = store.Create(ctx, "unique-b", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
			},
		},
		{
			name: "a Target that no longer exists fails the resolve and writes nothing",
			run: func(t *testing.T) {
				ids := seedTargets(t, "vanished", "us-east")
				rollout, err := store.Create(ctx, "vanished", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				_, err = store.ResolveTargets(ctx, rollout, []uuid.UUID{ids["us-east"], uuid.New()})
				require.ErrorIs(t, err, database.ErrConflict)
				fetched, err := store.Get(ctx, "vanished", "prod.01A.abc")
				require.NoError(t, err)
				require.Nil(t, fetched.TargetsResolvedAt)
				require.Empty(t, fetched.Targets)
				// So a second attempt, without the vanished Target, succeeds.
				resolved, err := store.ResolveTargets(ctx, fetched, []uuid.UUID{ids["us-east"]})
				require.NoError(t, err)
				require.Equal(t, []string{"us-east"}, targetNames(resolved))
			},
		},
		{
			name: "deleting a Target removes it from the snapshot",
			run: func(t *testing.T) {
				ids := seedTargets(t, "cascade", "us-east", "us-west")
				rollout, err := store.Create(ctx, "cascade", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				_, err = store.ResolveTargets(ctx, rollout, []uuid.UUID{ids["us-east"], ids["us-west"]})
				require.NoError(t, err)
				require.NoError(t, targets.Delete(ctx, "cascade", "us-west"))
				rollout, err = store.Get(ctx, "cascade", "prod.01A.abc")
				require.NoError(t, err)
				require.Equal(t, []string{"us-east"}, targetNames(rollout))
			},
		},
		{
			name: "the database rejects bad values",
			run: func(t *testing.T) {
				_, err := pool.Exec(ctx,
					`INSERT INTO promotion_rollouts (project_name, name, stage_name, number, freight_name, status_phase)
					 VALUES ('bad', 'a', 'prod', 1, 'f', 'Bogus')`,
				)
				databasetest.RequirePGError(t, err, "23514")
				_, err = pool.Exec(ctx,
					`INSERT INTO promotion_rollouts (project_name, name, stage_name, number, freight_name, max_concurrent)
					 VALUES ('bad', 'a', 'prod', 1, 'f', -1)`,
				)
				databasetest.RequirePGError(t, err, "23514")
				_, err = pool.Exec(ctx,
					`INSERT INTO promotion_rollouts (project_name, name, stage_name, number, freight_name)
					 VALUES ('bad', 'a', 'prod', 0, 'f')`,
				)
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "list is in creation order and can be narrowed to a Stage",
			run: func(t *testing.T) {
				ids := seedTargets(t, "list", "us-east")
				for _, name := range []string{"prod.01A.abc", "staging.01A.abc", "prod.01B.abc"} {
					stage := name[:len(name)-len(".01A.abc")]
					rollout, err := store.Create(ctx, "list", newRollout(stage, name))
					require.NoError(t, err)
					_, err = store.ResolveTargets(ctx, rollout, []uuid.UUID{ids["us-east"]})
					require.NoError(t, err)
				}
				all, err := store.List(ctx, "list", "")
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01A.abc", "staging.01A.abc", "prod.01B.abc"}, rolloutNames(all))
				for _, rollout := range all {
					require.Equal(t, []string{"us-east"}, targetNames(rollout))
				}
				prod, err := store.List(ctx, "list", "prod")
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc"}, rolloutNames(prod))
				require.EqualValues(t, 2, prod[1].Number)
				none, err := store.List(ctx, "nothing-here", "")
				require.NoError(t, err)
				require.Empty(t, none)
			},
		},
		{
			name: "get many returns what exists, by name",
			run: func(t *testing.T) {
				for _, name := range []string{"prod.01A.abc", "prod.01B.abc"} {
					_, err := store.Create(ctx, "many", newRollout("prod", name))
					require.NoError(t, err)
				}
				rollouts, err := store.GetMany(ctx, "many", []string{"prod.01B.abc", "missing", "prod.01A.abc"})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc"}, rolloutNames(rollouts))
				rollouts, err = store.GetMany(ctx, "many", nil)
				require.NoError(t, err)
				require.Nil(t, rollouts)
			},
		},
		{
			name: "open rollouts are those not in a terminal phase",
			run: func(t *testing.T) {
				phases := map[string]string{
					"prod.01A.abc": "Pending",
					"prod.01B.abc": "Running",
					"prod.01C.abc": "Succeeded",
					"prod.01D.abc": "Errored",
				}
				for name, phase := range phases {
					rollout, err := store.Create(ctx, "open", newRollout("prod", name))
					require.NoError(t, err)
					rollout.StatusPhase = phase
					_, err = store.UpdateStatus(ctx, "open", rollout.PromotionRolloutRow)
					require.NoError(t, err)
				}
				keys, err := store.ListOpen(ctx)
				require.NoError(t, err)
				var open []Key
				for _, key := range keys {
					if key.Project == "open" {
						open = append(open, key)
					}
				}
				require.Equal(t, []Key{
					{Project: "open", Name: "prod.01A.abc"},
					{Project: "open", Name: "prod.01B.abc"},
				}, open)
			},
		},
		{
			name: "update status advances the version and leaves Targets alone",
			run: func(t *testing.T) {
				ids := seedTargets(t, "update", "us-east")
				bare, err := store.Create(ctx, "update", newRollout("prod", "prod.01A.abc"))
				require.NoError(t, err)
				created, err := store.ResolveTargets(ctx, bare, []uuid.UUID{ids["us-east"]})
				require.NoError(t, err)
				now := time.Now().UTC().Truncate(time.Microsecond)
				row := created.PromotionRolloutRow
				row.StatusPhase = "Errored"
				row.StatusReason = "EnterpriseOnlyFeature"
				row.StatusMessage = "nope"
				row.StatusFinishedAt = &now
				updated, err := store.UpdateStatus(ctx, "update", row)
				require.NoError(t, err)
				require.True(t, updated.UpdatedAt.After(created.UpdatedAt))
				require.Equal(t, "Errored", updated.StatusPhase)
				require.Equal(t, "EnterpriseOnlyFeature", updated.StatusReason)
				require.Equal(t, "nope", updated.StatusMessage)
				require.True(t, updated.StatusFinishedAt.Equal(now))
				require.Nil(t, updated.StatusStartedAt)
				fetched, err := store.Get(ctx, "update", "prod.01A.abc")
				require.NoError(t, err)
				require.Equal(t, updated, fetched.PromotionRolloutRow)
				require.Equal(t, []string{"us-east"}, targetNames(fetched))

				// The version the first update was based on is now stale.
				_, err = store.UpdateStatus(ctx, "update", row)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "modified")

				// A different id says the rollout was replaced.
				row = updated
				row.ID = uuid.New()
				_, err = store.UpdateStatus(ctx, "update", row)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "recreated")

				// Without preconditions the update is unconditional.
				_, err = store.UpdateStatus(ctx, "update", database.PromotionRolloutRow{
					Name:        "prod.01A.abc",
					StatusPhase: "Failed",
				})
				require.NoError(t, err)

				_, err = store.UpdateStatus(ctx, "update", database.PromotionRolloutRow{
					Name:        "missing",
					StatusPhase: "Failed",
				})
				require.ErrorIs(t, err, database.ErrNotFound)

				_, err = store.UpdateStatus(ctx, "update", database.PromotionRolloutRow{
					Name:        "prod.01A.abc",
					StatusPhase: "Bogus",
				})
				require.ErrorIs(t, err, database.ErrInvalid)
			},
		},
		{
			name: "get reports a missing rollout",
			run: func(t *testing.T) {
				_, err := store.Get(ctx, "missing", "prod.01A.abc")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.run)
	}
}

func rolloutNames(rollouts []Rollout) []string {
	names := make([]string, len(rollouts))
	for i, rollout := range rollouts {
		names[i] = rollout.Name
	}
	return names
}
