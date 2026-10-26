//go:build integration && db

package promotionrolloutstore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
			target, err := targets.Create(ctx, project, database.TargetRow{
				Name:          name,
				TargetSetName: "fleet",
			})
			require.NoError(t, err)
			ids[name] = target.ID
		}
		return ids
	}
	// seedStage records a Stage as the mirror would, under the given UID.
	// Rollouts are numbered from the Stage's mirrored row, so one must exist
	// before a rollout for the Stage can be created.
	seedStage := func(t *testing.T, project, name, uid string) {
		t.Helper()
		require.NoError(t, queries.DeleteReplacedStage(ctx, database.DeleteReplacedStageParams{
			ProjectName: project,
			Name:        name,
			ID:          uid,
		}))
		require.NoError(t, queries.UpsertStage(ctx, database.UpsertStageParams{
			ID:          uid,
			ProjectName: project,
			Name:        name,
			CreatedAt:   time.Now(),
		}))
	}
	// Every case below but those about the mirror itself creates rollouts for
	// these Stages in a Project of its own.
	for _, project := range []string{
		"create", "numbers", "concurrent", "empty", "unique-a", "unique-b",
		"vanished", "cascade", "list", "latest", "many", "open", "start", "update",
	} {
		for _, stage := range []string{"prod", "staging"} {
			seedStage(t, project, stage, project+"/"+stage)
		}
	}
	newRollout := func(stage, name string) database.PromotionRolloutRow {
		return database.PromotionRolloutRow{
			Name:          name,
			StageName:     stage,
			FreightName:   "freight-1",
			MaxConcurrent: 2,
			MaxFailures:   1,
			CreatedBy:     "tester",
		}
	}

	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create returns the stored rollout with its Targets in name order",
			run: func(t *testing.T) {
				ids := seedTargets(t, "create", "us-west", "us-east")
				rollout, err := store.Create(
					ctx,
					"create",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-west"], ids["us-east"], ids["us-west"]},
				)
				require.NoError(t, err)
				require.Equal(t, "create", rollout.ProjectName)
				require.Equal(t, "prod.01A.abc", rollout.Name)
				require.Equal(t, "prod", rollout.StageName)
				require.EqualValues(t, 1, rollout.Number)
				require.Equal(t, "freight-1", rollout.FreightName)
				require.EqualValues(t, 2, rollout.MaxConcurrent)
				require.EqualValues(t, 1, rollout.MaxFailures)
				require.Equal(t, "tester", rollout.CreatedBy)
				require.NotEqual(t, uuid.Nil, rollout.ID)
				require.False(t, rollout.CreatedAt.IsZero())
				require.True(t, rollout.CreatedAt.Equal(rollout.UpdatedAt))
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
				first, err := store.Create(ctx, "numbers", newRollout("prod", "prod.01A.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 1, first.Number)
				second, err := store.Create(ctx, "numbers", newRollout("prod", "prod.01B.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 2, second.Number)
				other, err := store.Create(ctx, "numbers", newRollout("staging", "staging.01A.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 1, other.Number)

				// A duplicate name fails the whole statement, increment included.
				_, err = store.Create(ctx, "numbers", newRollout("prod", "prod.01B.abc"), nil)
				require.ErrorIs(t, err, database.ErrAlreadyExists)
				third, err := store.Create(ctx, "numbers", newRollout("prod", "prod.01C.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 3, third.Number)
			},
		},
		{
			name: "a Stage the mirror has not recorded yet is refused and nothing is written",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "unmirrored", newRollout("prod", "prod.01A.abc"), nil)
				require.ErrorIs(t, err, database.ErrNotMirrored)
				rollouts, err := store.List(ctx, "unmirrored", ListOptions{})
				require.NoError(t, err)
				require.Empty(t, rollouts)
			},
		},
		{
			name: "a recreated Stage numbers past the rollouts of its predecessor",
			run: func(t *testing.T) {
				seedStage(t, "recreated", "prod", "old-uid")
				for _, name := range []string{"prod.01A.abc", "prod.01B.abc"} {
					_, err := store.Create(ctx, "recreated", newRollout("prod", name), nil)
					require.NoError(t, err)
				}
				// The new Stage's row replaces the old one and starts its
				// counter over.
				seedStage(t, "recreated", "prod", "new-uid")
				rollout, err := store.Create(ctx, "recreated", newRollout("prod", "prod.01C.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 3, rollout.Number)
				rollout, err = store.Create(ctx, "recreated", newRollout("prod", "prod.01D.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 4, rollout.Number)
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
							ctx, "concurrent", newRollout("prod", "prod."+uuid.NewString()), nil,
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
			name: "a rollout created with no Targets has an empty list",
			run: func(t *testing.T) {
				rollout, err := store.Create(ctx, "empty", newRollout("prod", "prod.01A.abc"), nil)
				require.NoError(t, err)
				require.NotNil(t, rollout.Targets)
				require.Empty(t, rollout.Targets)
				fetched, err := store.Get(ctx, "empty", "prod.01A.abc")
				require.NoError(t, err)
				require.Empty(t, fetched.Targets)
			},
		},
		{
			name: "names are unique within a Project only",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "unique-a", newRollout("prod", "prod.01A.abc"), nil)
				require.NoError(t, err)
				_, err = store.Create(ctx, "unique-a", newRollout("prod", "prod.01A.abc"), nil)
				require.ErrorIs(t, err, database.ErrAlreadyExists)
				_, err = store.Create(ctx, "unique-b", newRollout("prod", "prod.01A.abc"), nil)
				require.NoError(t, err)
			},
		},
		{
			name: "a Target that no longer exists fails the create and writes nothing",
			run: func(t *testing.T) {
				ids := seedTargets(t, "vanished", "us-east")
				_, err := store.Create(
					ctx,
					"vanished",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-east"], uuid.New()},
				)
				require.ErrorIs(t, err, database.ErrConflict)
				_, err = store.Get(ctx, "vanished", "prod.01A.abc")
				require.ErrorIs(t, err, database.ErrNotFound)
				// The number is not used up either, so a second attempt,
				// without the vanished Target, is the Stage's first.
				rollout, err := store.Create(
					ctx,
					"vanished",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-east"]},
				)
				require.NoError(t, err)
				require.EqualValues(t, 1, rollout.Number)
				require.Equal(t, []string{"us-east"}, targetNames(rollout))
			},
		},
		{
			name: "deleting a Target removes it from the snapshot",
			run: func(t *testing.T) {
				ids := seedTargets(t, "cascade", "us-east", "us-west")
				_, err := store.Create(
					ctx,
					"cascade",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-east"], ids["us-west"]},
				)
				require.NoError(t, err)
				require.NoError(t, targets.Delete(ctx, "cascade", "us-west"))
				rollout, err := store.Get(ctx, "cascade", "prod.01A.abc")
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
					`INSERT INTO promotion_rollouts (project_name, name, stage_name, number, freight_name, max_failures)
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
			name: "lists are paged, newest first, and narrowed to a Stage or its Freight",
			run: func(t *testing.T) {
				ids := seedTargets(t, "list", "us-east")
				create := func(stage, name, freight string) Rollout {
					row := newRollout(stage, name)
					row.FreightName = freight
					rollout, err := store.Create(ctx, "list", row, []uuid.UUID{ids["us-east"]})
					require.NoError(t, err)
					return rollout
				}
				create("prod", "prod.01A.abc", "freight-1")
				create("staging", "staging.01A.abc", "freight-1")
				create("prod", "prod.01B.abc", "freight-2")
				create("prod", "prod.01C.abc", "freight-1")

				// A Project's rollouts, newest first, with their Targets.
				all, err := store.List(ctx, "list", ListOptions{})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01C.abc", "prod.01B.abc", "staging.01A.abc", "prod.01A.abc"}, rolloutNames(all))
				for _, rollout := range all {
					require.Equal(t, []string{"us-east"}, targetNames(rollout))
				}
				// A page at a time, continued from where the last one ended.
				page, err := store.List(ctx, "list", ListOptions{Limit: 2})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01C.abc", "prod.01B.abc"}, rolloutNames(page))
				at := CursorOf(page[1])
				rest, err := store.List(ctx, "list", ListOptions{Before: &at, Limit: 2})
				require.NoError(t, err)
				require.Equal(t, []string{"staging.01A.abc", "prod.01A.abc"}, rolloutNames(rest))

				// A Stage's rollouts, by number.
				prod, err := store.List(ctx, "list", ListOptions{Stage: "prod"})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01C.abc", "prod.01B.abc", "prod.01A.abc"}, rolloutNames(prod))
				require.EqualValues(t, 3, prod[0].Number)
				before := CursorOf(prod[0])
				older, err := store.List(ctx, "list", ListOptions{Stage: "prod", Before: &before})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01B.abc", "prod.01A.abc"}, rolloutNames(older))
				after, err := store.List(ctx, "list", ListOptions{Stage: "prod", After: &Cursor{Number: 1}})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01B.abc", "prod.01C.abc"}, rolloutNames(after))
				ofFreight, err := store.List(ctx, "list", ListOptions{Stage: "prod", Freight: "freight-1"})
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01C.abc", "prod.01A.abc"}, rolloutNames(ofFreight))

				// Options that do not go together.
				_, err = store.List(ctx, "list", ListOptions{Freight: "freight-1"})
				require.Error(t, err)
				_, err = store.List(ctx, "list", ListOptions{After: &Cursor{}})
				require.Error(t, err)
				_, err = store.List(ctx, "list", ListOptions{Stage: "prod", After: &Cursor{}, Freight: "freight-1"})
				require.Error(t, err)

				none, err := store.List(ctx, "nothing-here", ListOptions{})
				require.NoError(t, err)
				require.Empty(t, none)
			},
		},
		{
			name: "latest lists each Stage's open rollout and its newest finished one",
			run: func(t *testing.T) {
				create := func(stage, name string) Rollout {
					rollout, err := store.Create(ctx, "latest", newRollout(stage, name), nil)
					require.NoError(t, err)
					return rollout
				}
				setPhase := func(rollout Rollout, phase kargoapi.PromotionRolloutPhase) Rollout {
					row := rollout.PromotionRolloutRow
					row.StatusPhase = string(phase)
					row, err := store.UpdateStatus(ctx, "latest", row)
					require.NoError(t, err)
					rollout.PromotionRolloutRow = row
					return rollout
				}
				setPhase(create("prod", "prod.01A.abc"), kargoapi.PromotionRolloutPhaseSucceeded)
				running := setPhase(create("prod", "prod.01B.abc"), kargoapi.PromotionRolloutPhaseRunning)
				create("prod", "prod.01C.abc")
				create("prod", "prod.01D.abc")
				setPhase(create("staging", "staging.01A.abc"), kargoapi.PromotionRolloutPhaseFailed)

				// The Running rollout is the open one, over the Pending ones
				// behind it; the Stage with none open has only its finished one.
				latest, err := store.ListLatest(ctx, "latest", "")
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc", "staging.01A.abc"}, rolloutNames(latest))
				prod, err := store.ListLatest(ctx, "latest", "prod")
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01A.abc", "prod.01B.abc"}, rolloutNames(prod))

				// Once it finishes, the oldest Pending one is open, and it is
				// the newest finished.
				setPhase(running, kargoapi.PromotionRolloutPhaseSucceeded)
				prod, err = store.ListLatest(ctx, "latest", "prod")
				require.NoError(t, err)
				require.Equal(t, []string{"prod.01B.abc", "prod.01C.abc"}, rolloutNames(prod))

				none, err := store.ListLatest(ctx, "nothing-here", "")
				require.NoError(t, err)
				require.Empty(t, none)
			},
		},
		{
			name: "start admits one rollout per Stage at a time, in number order",
			run: func(t *testing.T) {
				create := func(stage, name string) Rollout {
					rollout, err := store.Create(ctx, "start", newRollout(stage, name), nil)
					require.NoError(t, err)
					return rollout
				}
				first := create("prod", "prod.01A.abc")
				second := create("prod", "prod.01B.abc")
				third := create("prod", "prod.01C.abc")
				other := create("staging", "staging.01A.abc")

				// Not next: an older rollout is still open.
				_, err := store.Start(ctx, second)
				require.ErrorIs(t, err, database.ErrConflict)
				started, err := store.Start(ctx, first)
				require.NoError(t, err)
				require.Equal(t, string(kargoapi.PromotionRolloutPhaseRunning), started.StatusPhase)
				require.NotNil(t, started.StatusStartedAt)
				require.True(t, started.UpdatedAt.After(first.UpdatedAt))
				// Not next: the Stage is promoting through another rollout.
				_, err = store.Start(ctx, second)
				require.ErrorIs(t, err, database.ErrConflict)
				// Not Pending any more.
				_, err = store.Start(ctx, first)
				require.ErrorIs(t, err, database.ErrConflict)
				// Another Stage's rollout is not held up.
				_, err = store.Start(ctx, other)
				require.NoError(t, err)

				// The index holds whatever writes a second Running row.
				row := third.PromotionRolloutRow
				row.StatusPhase = string(kargoapi.PromotionRolloutPhaseRunning)
				_, err = store.UpdateStatus(ctx, "start", row)
				require.ErrorIs(t, err, database.ErrConflict)

				// Once the round ends, the next in number order is admitted.
				row = started.PromotionRolloutRow
				row.StatusPhase = string(kargoapi.PromotionRolloutPhaseSucceeded)
				_, err = store.UpdateStatus(ctx, "start", row)
				require.NoError(t, err)
				_, err = store.Start(ctx, third)
				require.ErrorIs(t, err, database.ErrConflict)
				_, err = store.Start(ctx, second)
				require.NoError(t, err)
			},
		},
		{
			name: "get many returns what exists, by name",
			run: func(t *testing.T) {
				for _, name := range []string{"prod.01A.abc", "prod.01B.abc"} {
					_, err := store.Create(ctx, "many", newRollout("prod", name), nil)
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
					rollout, err := store.Create(ctx, "open", newRollout("prod", name), nil)
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
				created, err := store.Create(
					ctx,
					"update",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-east"]},
				)
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
