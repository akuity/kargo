//go:build integration && db

package promotionrolloutstore

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
	"github.com/akuity/kargo/pkg/database/targetstore"
	"github.com/akuity/kargo/pkg/event/fake"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// forProject returns the events published about the Project's rollouts, in
// the order they were published.
func forProject(t *testing.T, sender *fake.Sender, project string) []published {
	t.Helper()
	var events []published
	for _, e := range decodePublished(t, sender) {
		if e.event.Key.Project == project {
			events = append(events, e)
		}
	}
	return events
}

// inUTC returns the row with its times in UTC. A row read from Postgres has
// its times in time.Local, but one decoded from an event's JSON has them in
// UTC, so rows are compared in UTC to compare their instants, not their
// locations.
func inUTC(row database.PromotionRolloutRow) database.PromotionRolloutRow {
	row.CreatedAt = row.CreatedAt.UTC()
	row.UpdatedAt = row.UpdatedAt.UTC()
	if row.StatusStartedAt != nil {
		startedAt := row.StatusStartedAt.UTC()
		row.StatusStartedAt = &startedAt
	}
	if row.StatusFinishedAt != nil {
		finishedAt := row.StatusFinishedAt.UTC()
		row.StatusFinishedAt = &finishedAt
	}
	if row.AbortRequestedAt != nil {
		requestedAt := row.AbortRequestedAt.UTC()
		row.AbortRequestedAt = &requestedAt
	}
	return row
}

func TestStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	events := newRecordingSender()
	store := New(queries, nats.NewPublisherWithSender(events))
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
		"vanished", "cascade", "list", "latest", "many", "open", "running", "update",
		"statuses", "abort",
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

				// The creation is announced with the row as stored.
				published := forProject(t, events, "create")
				require.Len(t, published, 1)
				require.Equal(t, "akuity.kargo.internal.promotion.rollout.created", published[0].subject)
				require.Equal(t, nats.Created, published[0].event.Kind)
				require.Equal(t, Key{Project: "create", Name: "prod.01A.abc"}, published[0].event.Key)
				require.Equal(
					t,
					inUTC(rollout.PromotionRolloutRow),
					inUTC(*published[0].event.New),
				)
				require.Equal(t, []string{"us-east", "us-west"}, targetNames(rollout))
				require.Equal(t, ids["us-east"], rollout.Targets[0].TargetID)

				fetched, err := store.Get(ctx, "create", "prod.01A.abc")
				require.NoError(t, err)
				require.Equal(t, rollout, fetched)

				fetched, err = store.GetByNumber(ctx, "create", "prod", 1)
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
			name: "a rollout goes with its Stage, and a recreated Stage starts over",
			run: func(t *testing.T) {
				ids := seedTargets(t, "recreated", "us-east")
				seedStage(t, "recreated", "prod", "old-uid")
				for _, name := range []string{"prod.01A.abc", "prod.01B.abc"} {
					_, err := store.Create(
						ctx, "recreated", newRollout("prod", name), []uuid.UUID{ids["us-east"]},
					)
					require.NoError(t, err)
				}
				rollouts, err := store.List(ctx, "recreated", ListOptions{})
				require.NoError(t, err)
				require.Len(t, rollouts, 2)
				require.Equal(t, "old-uid", rollouts[0].StageID)
				// The new Stage's row replaces the old one, which takes the
				// old Stage's rollouts and their snapshots with it, so the
				// new Stage's history begins at 1.
				seedStage(t, "recreated", "prod", "new-uid")
				rollouts, err = store.List(ctx, "recreated", ListOptions{})
				require.NoError(t, err)
				require.Empty(t, rollouts)
				var snapshots int
				require.NoError(t, pool.QueryRow(ctx,
					"SELECT count(*) FROM promotion_rollout_targets WHERE target_id = $1",
					ids["us-east"],
				).Scan(&snapshots))
				require.Zero(t, snapshots)
				rollout, err := store.Create(ctx, "recreated", newRollout("prod", "prod.01C.abc"), nil)
				require.NoError(t, err)
				require.EqualValues(t, 1, rollout.Number)
				require.Equal(t, "new-uid", rollout.StageID)
				// Deleting the Stage's row, as the mirror does when the Stage
				// is deleted, deletes the rollouts the same way.
				require.NoError(t, queries.DeleteStageByName(ctx, database.DeleteStageByNameParams{
					ProjectName: "recreated",
					Name:        "prod",
				}))
				_, err = store.Get(ctx, "recreated", "prod.01C.abc")
				require.ErrorIs(t, err, database.ErrNotFound)
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
				seedStage(t, "bad", "prod", "bad/prod")
				const insert = `INSERT INTO promotion_rollouts
					(project_name, name, stage_id, stage_name, number, freight_name, %s)
					VALUES ('bad', 'a', 'bad/prod', 'prod', %s, 'f', %s)`
				for _, bad := range []struct{ column, number, value string }{
					{column: "status_phase", number: "1", value: "'Bogus'"},
					{column: "max_concurrent", number: "1", value: "-1"},
					{column: "max_failures", number: "1", value: "-1"},
					{column: "max_failures", number: "0", value: "0"},
				} {
					_, err := pool.Exec(ctx, fmt.Sprintf(insert, bad.column, bad.number, bad.value))
					databasetest.RequirePGError(t, err, "23514")
				}
				// A rollout must reference a Stage's row, and the one it
				// references must exist.
				_, err := pool.Exec(ctx,
					`INSERT INTO promotion_rollouts (project_name, name, stage_name, number, freight_name)
					 VALUES ('bad', 'a', 'prod', 1, 'f')`,
				)
				databasetest.RequirePGError(t, err, "23502")
				_, err = pool.Exec(ctx,
					`INSERT INTO promotion_rollouts (project_name, name, stage_id, stage_name, number, freight_name)
					 VALUES ('bad', 'a', 'bad/absent', 'prod', 1, 'f')`,
				)
				databasetest.RequirePGError(t, err, "23503")
			},
		},
		{
			name: "a Stage's row keys its rollouts' numbers and its Running one",
			run: func(t *testing.T) {
				seedStage(t, "keys", "prod", "keys/prod")
				insert := func(name, stage string, number int, phase string) error {
					_, err := pool.Exec(ctx,
						`INSERT INTO promotion_rollouts
						 (project_name, name, stage_id, stage_name, number, freight_name, status_phase)
						 VALUES ('keys', $1, 'keys/prod', $2, $3, 'f', $4)`,
						name, stage, number, phase,
					)
					return err
				}
				requireViolation := func(t *testing.T, err error, code, constraint string) {
					t.Helper()
					var pgErr *pgconn.PgError
					require.ErrorAs(t, err, &pgErr)
					require.Equal(t, code, pgErr.Code)
					require.Equal(t, constraint, pgErr.ConstraintName)
				}
				require.NoError(t, insert("a", "prod", 1, "Running"))
				// Two rollouts of the Stage's row cannot both be Running...
				requireViolation(t, insert("b", "prod", 2, "Running"), "23505", runningConstraint)
				// ...nor share a number.
				requireViolation(
					t, insert("b", "prod", 1, "Pending"), "23505", "promotion_rollouts_stage_number_key",
				)
				require.NoError(t, insert("b", "prod", 2, "Pending"))
				// A rollout carries the names of the Stage whose row it
				// references.
				requireViolation(
					t, insert("c", "staging", 3, "Pending"), "23503", "promotion_rollouts_stage_fkey",
				)
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
			name: "one rollout per Stage can be Running at a time",
			run: func(t *testing.T) {
				create := func(stage, name string) Rollout {
					rollout, err := store.Create(ctx, "running", newRollout(stage, name), nil)
					require.NoError(t, err)
					return rollout
				}
				// setPhase writes the phase without preconditions, so a row may
				// be written more than once.
				setPhase := func(name string, phase kargoapi.PromotionRolloutPhase) error {
					_, err := store.UpdateStatus(ctx, "running", database.PromotionRolloutRow{
						Name:        name,
						StatusPhase: string(phase),
					})
					return err
				}
				create("prod", "prod.01A.abc")
				create("prod", "prod.01B.abc")
				create("staging", "staging.01A.abc")

				require.NoError(t, setPhase("prod.01A.abc", kargoapi.PromotionRolloutPhaseRunning))
				// The index holds whatever writes a second Running row of the
				// Stage.
				err := setPhase("prod.01B.abc", kargoapi.PromotionRolloutPhaseRunning)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "another PromotionRollout of the Stage is Running")
				// Another Stage's rollout is not held up.
				require.NoError(t, setPhase("staging.01A.abc", kargoapi.PromotionRolloutPhaseRunning))

				// Once the first finishes, the second may run.
				require.NoError(t, setPhase("prod.01A.abc", kargoapi.PromotionRolloutPhaseSucceeded))
				require.NoError(t, setPhase("prod.01B.abc", kargoapi.PromotionRolloutPhaseRunning))
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

				// The update is announced with the row before and after it.
				published := forProject(t, events, "update")
				require.Len(t, published, 2)
				require.Equal(t, nats.Created, published[0].event.Kind)
				require.Equal(t, "akuity.kargo.internal.promotion.rollout.updated", published[1].subject)
				require.Equal(t, nats.Updated, published[1].event.Kind)
				require.Equal(t, Key{Project: "update", Name: "prod.01A.abc"}, published[1].event.Key)
				require.Equal(
					t,
					inUTC(created.PromotionRolloutRow),
					inUTC(*published[1].event.Old),
				)
				require.Equal(t, inUTC(updated), inUTC(*published[1].event.New))

				// The version the first update was based on is now stale, and
				// a refused update announces nothing.
				_, err = store.UpdateStatus(ctx, "update", row)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "modified")
				require.Len(t, forProject(t, events, "update"), 2)

				// A different id says the rollout was replaced.
				row = updated
				row.ID = uuid.New()
				_, err = store.UpdateStatus(ctx, "update", row)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "recreated")

				// Without preconditions the update is unconditional, and
				// announced from the row as it was just before.
				_, err = store.UpdateStatus(ctx, "update", database.PromotionRolloutRow{
					Name:        "prod.01A.abc",
					StatusPhase: "Failed",
				})
				require.NoError(t, err)
				published = forProject(t, events, "update")
				require.Len(t, published, 3)
				require.Equal(t, nats.Updated, published[2].event.Kind)
				require.Equal(t, "Errored", published[2].event.Old.StatusPhase)
				require.Equal(t, "Failed", published[2].event.New.StatusPhase)

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
			name: "abort ends a Pending rollout and only records the request on a Running one",
			run: func(t *testing.T) {
				ids := seedTargets(t, "abort", "us-east")
				pending, err := store.Create(
					ctx,
					"abort",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-east"]},
				)
				require.NoError(t, err)
				running, err := store.Create(
					ctx,
					"abort",
					newRollout("staging", "staging.01A.abc"),
					[]uuid.UUID{ids["us-east"]},
				)
				require.NoError(t, err)
				started := time.Now().UTC().Truncate(time.Microsecond)
				row := running.PromotionRolloutRow
				row.StatusPhase = "Running"
				row.StatusReason = "Promoting"
				row.StatusMessage = "Waiting on 1 of 1 Promotion to complete"
				row.StatusStartedAt = &started
				runningRow, err := store.UpdateStatus(ctx, "abort", row)
				require.NoError(t, err)

				// A Pending rollout has started nothing, so it ends at once.
				aborted, err := store.Abort(ctx, "abort", "prod", 1, "alice")
				require.NoError(t, err)
				require.Equal(t, "Aborted", aborted.StatusPhase)
				require.Equal(t, ReasonAborted, aborted.StatusReason)
				require.Equal(t, "Aborted by alice before it started", aborted.StatusMessage)
				require.Nil(t, aborted.StatusStartedAt)
				require.NotNil(t, aborted.StatusFinishedAt)
				require.NotNil(t, aborted.AbortRequestedAt)
				require.Equal(t, "alice", aborted.AbortRequestedBy)
				require.True(t, aborted.UpdatedAt.After(pending.UpdatedAt))

				// The abort is announced with the row before and after it,
				// after the two creations and the start of the Running one.
				published := forProject(t, events, "abort")
				require.Len(t, published, 4)
				require.Equal(t, nats.Updated, published[3].event.Kind)
				require.Equal(t, Key{Project: "abort", Name: "prod.01A.abc"}, published[3].event.Key)
				require.Equal(t, inUTC(pending.PromotionRolloutRow), inUTC(*published[3].event.Old))
				require.Equal(t, inUTC(aborted), inUTC(*published[3].event.New))

				// The reconciler's claim of the rollout, based on the row as it
				// read it before the abort, is refused.
				claim := pending.PromotionRolloutRow
				claim.StatusPhase = "Running"
				_, err = store.UpdateStatus(ctx, "abort", claim)
				require.ErrorIs(t, err, database.ErrConflict)

				// A finished rollout cannot be aborted, this one included.
				_, err = store.Abort(ctx, "abort", "prod", 1, "bob")
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "already finished as Aborted")
				// Neither refusal announces anything.
				require.Len(t, forProject(t, events, "abort"), 4)

				// A Running rollout keeps its status, which the reconciler
				// owns; only the request is recorded.
				requested, err := store.Abort(ctx, "abort", "staging", 1, "alice")
				require.NoError(t, err)
				require.Equal(t, "Running", requested.StatusPhase)
				require.Equal(t, "Promoting", requested.StatusReason)
				require.Equal(t, runningRow.StatusMessage, requested.StatusMessage)
				require.True(t, requested.StatusStartedAt.Equal(started))
				require.Nil(t, requested.StatusFinishedAt)
				require.NotNil(t, requested.AbortRequestedAt)
				require.Equal(t, "alice", requested.AbortRequestedBy)
				require.True(t, requested.UpdatedAt.After(runningRow.UpdatedAt))
				published = forProject(t, events, "abort")
				require.Len(t, published, 5)
				require.Equal(t, nats.Updated, published[4].event.Kind)
				require.Equal(t, Key{Project: "abort", Name: "staging.01A.abc"}, published[4].event.Key)
				require.Equal(t, inUTC(runningRow), inUTC(*published[4].event.Old))
				require.Equal(t, inUTC(requested), inUTC(*published[4].event.New))

				// A second request keeps the first.
				again, err := store.Abort(ctx, "abort", "staging", 1, "bob")
				require.NoError(t, err)
				require.Equal(t, "alice", again.AbortRequestedBy)
				require.True(t, again.AbortRequestedAt.Equal(*requested.AbortRequestedAt))
				// It changes nothing, so it is not announced.
				require.Len(t, forProject(t, events, "abort"), 5)

				// The reconciler's status writes are unaffected by the request,
				// and leave it in place.
				row = again
				row.StatusPhase = "Aborted"
				finished, err := store.UpdateStatus(ctx, "abort", row)
				require.NoError(t, err)
				require.Equal(t, "alice", finished.AbortRequestedBy)
				require.NotNil(t, finished.AbortRequestedAt)

				open, err := store.ListOpen(ctx)
				require.NoError(t, err)
				for _, key := range open {
					require.NotEqual(t, "abort", key.Project)
				}

				_, err = store.Abort(ctx, "abort", "prod", 99, "alice")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "target statuses report the newest mirrored Promotion to each Target",
			run: func(t *testing.T) {
				ids := seedTargets(t, "statuses", "us-east", "us-west", "eu-west")
				rollout, err := store.Create(
					ctx,
					"statuses",
					newRollout("prod", "prod.01A.abc"),
					[]uuid.UUID{ids["us-east"], ids["us-west"], ids["eu-west"]},
				)
				require.NoError(t, err)
				other, err := store.Create(ctx, "statuses", newRollout("prod", "prod.01B.abc"), nil)
				require.NoError(t, err)

				// No Target has a Promotion yet.
				statuses, err := store.ListTargetStatuses(ctx, []Rollout{rollout, other})
				require.NoError(t, err)
				require.Empty(t, statuses)

				promotion := func(name, target string, of Rollout, phase string) database.UpsertPromotionParams {
					params := database.UpsertPromotionParams{
						ID:                 name,
						ProjectName:        "statuses",
						Name:               name,
						StageName:          "prod",
						FreightName:        "freight-1",
						PromotionRolloutID: &of.ID,
						Phase:              phase,
						CreatedAt:          time.Now(),
					}
					if target != "" {
						params.TargetName = &target
					}
					return params
				}
				for _, params := range []database.UpsertPromotionParams{
					// us-west: one Running Promotion.
					promotion("prod.us-west.01A.abc", "us-west", rollout, "Running"),
					// us-east: promoted to twice; the newest name wins.
					promotion("prod.us-east.01C.abc", "us-east", rollout, "Succeeded"),
					promotion("prod.us-east.01A.abc", "us-east", rollout, "Failed"),
					// eu-west: not picked up by the controller yet.
					promotion("prod.eu-west.01A.abc", "eu-west", rollout, ""),
					// Ignored: to no Target. Of another rollout: reported under it.
					promotion("prod.01D.abc", "", rollout, "Succeeded"),
					promotion("prod.us-east.01E.abc", "us-east", other, "Succeeded"),
				} {
					_, err = queries.UpsertPromotion(ctx, params)
					require.NoError(t, err)
				}

				row := func(of Rollout, target, name, phase string) database.ListPromotionRolloutTargetStatusesRow {
					return database.ListPromotionRolloutTargetStatusesRow{
						PromotionRolloutID: of.ID,
						TargetName:         target,
						PromotionName:      name,
						Phase:              phase,
					}
				}
				statuses, err = store.ListTargetStatuses(ctx, []Rollout{rollout, other})
				require.NoError(t, err)
				require.Equal(t, map[uuid.UUID][]database.ListPromotionRolloutTargetStatusesRow{
					rollout.ID: {
						row(rollout, "eu-west", "prod.eu-west.01A.abc", "Pending"),
						row(rollout, "us-east", "prod.us-east.01C.abc", "Succeeded"),
						row(rollout, "us-west", "prod.us-west.01A.abc", "Running"),
					},
					other.ID: {row(other, "us-east", "prod.us-east.01E.abc", "Succeeded")},
				}, statuses)

				targetStatuses, summary := TargetStatuses(statuses[rollout.ID])
				require.Len(t, targetStatuses, 3)
				require.Equal(t, &kargoapi.PromotionRolloutSummary{Pending: 1, Running: 1, Succeeded: 1}, summary)

				// Only the rollouts asked about are reported.
				statuses, err = store.ListTargetStatuses(ctx, []Rollout{other})
				require.NoError(t, err)
				require.Len(t, statuses, 1)
				require.Len(t, statuses[other.ID], 1)
				statuses, err = store.ListTargetStatuses(ctx, nil)
				require.NoError(t, err)
				require.Empty(t, statuses)
			},
		},
		{
			name: "get reports a missing rollout",
			run: func(t *testing.T) {
				_, err := store.Get(ctx, "missing", "prod.01A.abc")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "get by number reports a number the Stage has not reached",
			run: func(t *testing.T) {
				_, err := store.GetByNumber(ctx, "create", "prod", 99)
				require.ErrorIs(t, err, database.ErrNotFound)
				// Numbers are per Stage: the other Stage has no #1.
				_, err = store.GetByNumber(ctx, "create", "staging", 1)
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

func TestStoreFreightIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	events := newRecordingSender()
	store := New(queries, nats.NewPublisherWithSender(events))
	ctx := context.Background()

	origin := func(name string) kargoapi.FreightOrigin {
		return kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: name}
	}
	images, config, charts := origin("images"), origin("config"), origin("charts")
	// freight returns a piece of Freight from the origin with contents of
	// every kind, all of them particular to it.
	freight := func(name string, from kargoapi.FreightOrigin) kargoapi.FreightReference {
		return kargoapi.FreightReference{
			Name:   name,
			Origin: from,
			Commits: []kargoapi.GitCommit{{
				RepoURL: "https://example.com/" + from.Name + ".git",
				ID:      "commit-of-" + name,
				Branch:  "main",
				Message: "a commit\n\nwith a body, \"quotes\" and a snowman ☃",
			}},
			Images: []kargoapi.Image{{
				RepoURL:     "example.com/" + from.Name,
				Tag:         name,
				Digest:      "sha256:" + name,
				Annotations: map[string]string{"org.opencontainers.image.source": "example.com"},
			}},
			Charts: []kargoapi.Chart{{
				RepoURL: "oci://example.com/charts",
				Name:    from.Name,
				Version: "1.0.0+" + name,
			}},
			Artifacts: []kargoapi.ArtifactReference{{
				ArtifactType:     "application/vnd.example",
				SubscriptionName: "sub",
				Version:          name,
				Metadata:         &apiextensionsv1.JSON{Raw: []byte(`{"size":42,"labels":{"b":"2","a":"1"}}`)},
			}},
		}
	}
	// requireCollection requires the rollout's collection to hold exactly the
	// given Freight, contents included. The database keeps the contents as
	// they were written, so even an artifact's opaque metadata comes back
	// byte for byte.
	requireCollection := func(t *testing.T, rollout Rollout, refs ...kargoapi.FreightReference) {
		t.Helper()
		got, err := store.GetFreightCollection(ctx, rollout.ID)
		require.NoError(t, err)
		want := &kargoapi.FreightCollection{}
		if len(refs) > 0 {
			want.UpdateOrPush(refs...)
		}
		require.Equal(t, want, got)
	}
	countRows := func(t *testing.T, rollout Rollout) int {
		t.Helper()
		var count int
		require.NoError(t, pool.QueryRow(ctx,
			"SELECT count(*) FROM promotion_rollout_freight WHERE promotion_rollout_id = $1",
			rollout.ID,
		).Scan(&count))
		return count
	}
	seedStage := func(t *testing.T, project, name string) {
		t.Helper()
		require.NoError(t, queries.UpsertStage(ctx, database.UpsertStageParams{
			ID:          project + "/" + name,
			ProjectName: project,
			Name:        name,
			CreatedAt:   time.Now(),
		}))
	}
	// create creates a rollout of the Freight for the Project's Stage.
	create := func(t *testing.T, project, stage, name string, ref kargoapi.FreightReference) Rollout {
		t.Helper()
		rollout, err := store.Create(ctx, project, database.PromotionRolloutRow{
			Name:        name,
			StageName:   stage,
			FreightName: ref.Name,
		}, nil)
		require.NoError(t, err)
		return rollout
	}
	// finish moves the rollout to the phase it ends in.
	finish := func(t *testing.T, rollout Rollout, phase kargoapi.PromotionRolloutPhase) {
		t.Helper()
		_, err := store.UpdateStatus(ctx, rollout.ProjectName, database.PromotionRolloutRow{
			Name:        rollout.Name,
			StatusPhase: string(phase),
		})
		require.NoError(t, err)
	}
	// promote creates a rollout of the Freight for the Project's prod Stage,
	// starts it while the Stage requests Freight from the given origins, and
	// finishes it in the given phase.
	promote := func(
		t *testing.T,
		project, name string,
		ref kargoapi.FreightReference,
		phase kargoapi.PromotionRolloutPhase,
		requested ...kargoapi.FreightOrigin,
	) Rollout {
		t.Helper()
		rollout := create(t, project, "prod", name, ref)
		_, err := store.Start(ctx, project, rollout.PromotionRolloutRow, ref, requested)
		require.NoError(t, err)
		finish(t, rollout, phase)
		return rollout
	}
	const succeeded = kargoapi.PromotionRolloutPhaseSucceeded

	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "a Stage's first rollout records the Freight it promotes as it starts",
			run: func(t *testing.T) {
				const project = "first"
				seedStage(t, project, "prod")
				ref := freight("images-1", images)
				rollout := create(t, project, "prod", "prod.01A", ref)
				// A rollout that has not started has no collection yet.
				requireCollection(t, rollout)

				now := time.Now().UTC().Truncate(time.Microsecond)
				row := rollout.PromotionRolloutRow
				row.StatusReason = "Started"
				row.StatusMessage = "promoting"
				row.StatusStartedAt = &now
				// Whatever phase the argument carries, a start is to Running.
				row.StatusPhase = "Bogus"
				started, err := store.Start(ctx, project, row, ref, []kargoapi.FreightOrigin{images, config})
				require.NoError(t, err)
				require.Equal(t, rollout.ID, started.ID)
				require.Equal(t, "Running", started.StatusPhase)
				require.Equal(t, "Started", started.StatusReason)
				require.Equal(t, "promoting", started.StatusMessage)
				require.True(t, started.StatusStartedAt.Equal(now))
				require.Nil(t, started.StatusFinishedAt)
				require.True(t, started.UpdatedAt.After(rollout.UpdatedAt))
				fetched, err := store.Get(ctx, project, "prod.01A")
				require.NoError(t, err)
				require.Equal(t, started, fetched.PromotionRolloutRow)
				// The start is announced as any status update is.
				published := forProject(t, events, project)
				require.Len(t, published, 2)
				require.Equal(t, nats.Updated, published[1].event.Kind)
				require.Equal(t, inUTC(rollout.PromotionRolloutRow), inUTC(*published[1].event.Old))
				require.Equal(t, inUTC(started), inUTC(*published[1].event.New))

				requireCollection(t, rollout, ref)
				// The origin and the Freight's name are columns as well.
				var kind, name, freightName string
				require.NoError(t, pool.QueryRow(ctx,
					`SELECT origin_kind, origin_name, freight_name FROM promotion_rollout_freight
					 WHERE promotion_rollout_id = $1`,
					rollout.ID,
				).Scan(&kind, &name, &freightName))
				require.Equal(t, []string{"Warehouse", "images", "images-1"}, []string{kind, name, freightName})
			},
		},
		{
			name: "Freight whose contents hold a NUL is recorded and read back as it was",
			run: func(t *testing.T) {
				const project = "nul"
				seedStage(t, project, "prod")
				// Go encodes a NUL in a string as the escape \u0000, which
				// JSONB rejects.
				ref := freight("images-1", images)
				ref.Commits[0].Message = "a commit with a NUL\x00in it"
				ref.Artifacts[0].Metadata = &apiextensionsv1.JSON{Raw: []byte(`{"blob":"a\u0000b"}`)}
				rollout := create(t, project, "prod", "prod.01A", ref)
				_, err := store.Start(ctx, project, rollout.PromotionRolloutRow, ref, []kargoapi.FreightOrigin{images})
				require.NoError(t, err)
				requireCollection(t, rollout, ref)
				// And it is carried forward as it was.
				finish(t, rollout, succeeded)
				next := promote(t, project, "prod.01B", freight("config-1", config), succeeded, images, config)
				requireCollection(t, next, ref, freight("config-1", config))
			},
		},
		{
			name: "a rollout of another origin carries the first forward, contents intact",
			run: func(t *testing.T) {
				const project = "second"
				seedStage(t, project, "prod")
				first := promote(t, project, "prod.01A", freight("images-1", images), succeeded, images, config)
				second := promote(t, project, "prod.01B", freight("config-1", config), succeeded, images, config)
				requireCollection(t, second, freight("images-1", images), freight("config-1", config))
				// The row carried forward is a copy of the one it came from.
				var same bool
				require.NoError(t, pool.QueryRow(ctx,
					`SELECT copied.reference::text = original.reference::text
					 	AND copied.freight_name = original.freight_name
					 FROM promotion_rollout_freight AS copied, promotion_rollout_freight AS original
					 WHERE copied.promotion_rollout_id = $1 AND original.promotion_rollout_id = $2
					 	AND copied.origin_name = 'images' AND original.origin_name = 'images'`,
					second.ID, first.ID,
				).Scan(&same))
				require.True(t, same)
				// Read again, it is the same collection in the same order.
				once, err := store.GetFreightCollection(ctx, second.ID)
				require.NoError(t, err)
				again, err := store.GetFreightCollection(ctx, second.ID)
				require.NoError(t, err)
				require.Equal(t, once, again)
				require.Equal(t, []string{"config-1", "images-1"}, freightNames(once))
			},
		},
		{
			name: "an origin promoted again is replaced in the new rollout and kept in the old ones",
			run: func(t *testing.T) {
				const project = "again"
				seedStage(t, project, "prod")
				first := promote(t, project, "prod.01A", freight("images-1", images), succeeded, images, config)
				second := promote(t, project, "prod.01B", freight("config-1", config), succeeded, images, config)
				third := promote(t, project, "prod.01C", freight("images-2", images), succeeded, images, config)
				requireCollection(t, third, freight("images-2", images), freight("config-1", config))
				requireCollection(t, second, freight("images-1", images), freight("config-1", config))
				requireCollection(t, first, freight("images-1", images))
			},
		},
		{
			name: "an origin the Stage no longer requests is not carried forward",
			run: func(t *testing.T) {
				const project = "unrequested"
				seedStage(t, project, "prod")
				promote(t, project, "prod.01A", freight("images-1", images), succeeded, images, config)
				promote(t, project, "prod.01B", freight("config-1", config), succeeded, images, config)
				// The Stage has swapped images for charts.
				third := promote(t, project, "prod.01C", freight("charts-1", charts), succeeded, config, charts)
				requireCollection(t, third, freight("config-1", config), freight("charts-1", charts))
				// What a rollout promotes is recorded whether or not its
				// origin is among those given, and with none given nothing
				// else is.
				fourth := promote(t, project, "prod.01D", freight("images-2", images), succeeded)
				requireCollection(t, fourth, freight("images-2", images))
				// An origin missing from the previous rollout stays missing,
				// though an older rollout recorded it.
				fifth := promote(
					t, project, "prod.01E", freight("images-3", images), succeeded, images, config, charts,
				)
				requireCollection(t, fifth, freight("images-3", images))
			},
		},
		{
			name: "the rollout before is the newest that recorded a collection, however it finished",
			run: func(t *testing.T) {
				const project = "previous"
				seedStage(t, project, "prod")
				seedStage(t, project, "staging")
				all := []kargoapi.FreightOrigin{images, config, charts}
				promote(t, project, "prod.01A", freight("images-1", images), succeeded, all...)
				promote(t, project, "prod.01B", freight("config-1", config), succeeded, all...)
				// A rollout that fails recorded what it set out to promote,
				// and the next rollout carries that forward, as a Stage's
				// next Promotion does after one that failed.
				failed := promote(
					t, project, "prod.01C", freight("config-2", config), kargoapi.PromotionRolloutPhaseFailed, all...,
				)
				requireCollection(t, failed, freight("images-1", images), freight("config-2", config))
				afterFailed := promote(
					t, project, "prod.01D", freight("charts-1", charts), kargoapi.PromotionRolloutPhaseErrored, all...,
				)
				requireCollection(
					t, afterFailed,
					freight("images-1", images), freight("config-2", config), freight("charts-1", charts),
				)
				// Aborted before it started, so it recorded nothing, and it
				// is passed over for the one before it.
				aborted := create(t, project, "prod", "prod.01E", freight("charts-2", charts))
				finish(t, aborted, kargoapi.PromotionRolloutPhaseAborted)
				requireCollection(t, aborted)
				// Another Stage's rollouts are not this Stage's history.
				staging := create(t, project, "staging", "staging.01A", freight("charts-3", charts))
				_, err := store.Start(ctx, project, staging.PromotionRolloutRow, freight("charts-3", charts), all)
				require.NoError(t, err)
				finish(t, staging, succeeded)
				requireCollection(t, staging, freight("charts-3", charts))

				next := promote(t, project, "prod.01F", freight("images-2", images), succeeded, all...)
				requireCollection(
					t, next,
					freight("images-2", images), freight("config-2", config), freight("charts-1", charts),
				)

				// Each took the rollout numbered just below it but for the
				// one after the rollout that recorded nothing.
				second, err := store.Get(ctx, project, "prod.01B")
				require.NoError(t, err)
				requireCollection(t, second, freight("images-1", images), freight("config-1", config))
			},
		},
		{
			name: "a rollout's collection is deleted with it",
			run: func(t *testing.T) {
				const project = "deleted"
				seedStage(t, project, "prod")
				first := promote(t, project, "prod.01A", freight("images-1", images), succeeded, images, config)
				second := promote(t, project, "prod.01B", freight("config-1", config), succeeded, images, config)
				require.Equal(t, 1, countRows(t, first))
				require.Equal(t, 2, countRows(t, second))
				// Deleting a rollout takes its rows and leaves the copies
				// another rollout made of them.
				_, err := pool.Exec(ctx, "DELETE FROM promotion_rollouts WHERE id = $1", first.ID)
				require.NoError(t, err)
				require.Zero(t, countRows(t, first))
				requireCollection(t, second, freight("images-1", images), freight("config-1", config))
				// A Stage's deletion deletes its rollouts, and so their rows.
				require.NoError(t, queries.DeleteStageByName(ctx, database.DeleteStageByNameParams{
					ProjectName: project,
					Name:        "prod",
				}))
				require.Zero(t, countRows(t, second))
				requireCollection(t, second)
			},
		},
		{
			name: "a rollout is started once, and its collection is never rewritten",
			run: func(t *testing.T) {
				const project = "twice"
				seedStage(t, project, "prod")
				promote(t, project, "prod.01A", freight("images-1", images), succeeded, images, config)
				ref := freight("config-1", config)
				rollout := create(t, project, "prod", "prod.01B", ref)
				started, err := store.Start(
					ctx, project, rollout.PromotionRolloutRow, ref, []kargoapi.FreightOrigin{config},
				)
				require.NoError(t, err)
				requireCollection(t, rollout, ref)

				// A second start is refused, with or without preconditions,
				// and changes nothing.
				other := ref
				other.Images = nil
				for _, row := range []database.PromotionRolloutRow{started, {Name: rollout.Name}} {
					_, err = store.Start(ctx, project, row, other, []kargoapi.FreightOrigin{images, config})
					require.ErrorIs(t, err, database.ErrConflict)
					require.ErrorContains(t, err, "is Running, not Pending")
				}
				requireCollection(t, rollout, ref)
				fetched, err := store.Get(ctx, project, rollout.Name)
				require.NoError(t, err)
				require.Equal(t, started, fetched.PromotionRolloutRow)

				// Even a rollout written back to Pending, which nothing
				// should do, keeps the collection it has when started again.
				finish(t, rollout, kargoapi.PromotionRolloutPhasePending)
				_, err = store.Start(
					ctx, project, database.PromotionRolloutRow{Name: rollout.Name}, other,
					[]kargoapi.FreightOrigin{images, config},
				)
				require.NoError(t, err)
				requireCollection(t, rollout, ref)
				require.Equal(t, 1, countRows(t, rollout))

				// Freight that names no origin is refused even then, though
				// no row would be inserted for the database to reject.
				finish(t, rollout, kargoapi.PromotionRolloutPhasePending)
				noOrigin := ref
				noOrigin.Origin = kargoapi.FreightOrigin{}
				_, err = store.Start(
					ctx, project, database.PromotionRolloutRow{Name: rollout.Name}, noOrigin, nil,
				)
				require.ErrorIs(t, err, database.ErrInvalid)
				require.ErrorContains(t, err, "has no origin")
				fetched, err = store.Get(ctx, project, rollout.Name)
				require.NoError(t, err)
				require.Equal(t, "Pending", fetched.StatusPhase)
				requireCollection(t, rollout, ref)
			},
		},
		{
			name: "concurrent starts of one rollout record one collection",
			run: func(t *testing.T) {
				const project = "concurrent"
				seedStage(t, project, "prod")
				ref := freight("images-1", images)
				rollout := create(t, project, "prod", "prod.01A", ref)
				const n = 8
				var wg sync.WaitGroup
				errs := make([]error, n)
				for i := range n {
					wg.Go(func() {
						_, errs[i] = store.Start(
							ctx, project, database.PromotionRolloutRow{Name: rollout.Name}, ref,
							[]kargoapi.FreightOrigin{images},
						)
					})
				}
				wg.Wait()
				var started int
				for _, err := range errs {
					if err == nil {
						started++
						continue
					}
					require.ErrorIs(t, err, database.ErrConflict)
				}
				require.Equal(t, 1, started)
				requireCollection(t, rollout, ref)
			},
		},
		{
			name: "a rollout with no rows has an empty collection and is passed over",
			run: func(t *testing.T) {
				const project = "none"
				seedStage(t, project, "prod")
				// As a rollout started before collections were recorded: moved
				// to Running by a status update, which records no Freight.
				old := create(t, project, "prod", "prod.01A", freight("images-1", images))
				finish(t, old, kargoapi.PromotionRolloutPhaseRunning)
				collection, err := store.GetFreightCollection(ctx, old.ID)
				require.NoError(t, err)
				require.Equal(t, &kargoapi.FreightCollection{}, collection)
				finish(t, old, succeeded)
				require.Zero(t, countRows(t, old))
				// An id that is no rollout's reads the same way.
				collection, err = store.GetFreightCollection(ctx, uuid.New())
				require.NoError(t, err)
				require.Equal(t, &kargoapi.FreightCollection{}, collection)
				// The next rollout has nothing to carry forward: no rollout
				// before it recorded anything.
				next := promote(t, project, "prod.01B", freight("config-1", config), succeeded, images, config)
				requireCollection(t, next, freight("config-1", config))
				// Another rollout with no rows does not come between that one
				// and the one after it.
				old = create(t, project, "prod", "prod.01C", freight("images-2", images))
				finish(t, old, kargoapi.PromotionRolloutPhaseRunning)
				finish(t, old, succeeded)
				after := promote(t, project, "prod.01D", freight("images-3", images), succeeded, images, config)
				requireCollection(t, after, freight("images-3", images), freight("config-1", config))
			},
		},
		{
			name: "a start that is refused writes nothing",
			run: func(t *testing.T) {
				const project = "refused"
				seedStage(t, project, "prod")
				ref := freight("images-1", images)
				rollout := create(t, project, "prod", "prod.01A", ref)
				requirePending := func(t *testing.T, rollout Rollout) {
					t.Helper()
					fetched, err := store.Get(ctx, project, rollout.Name)
					require.NoError(t, err)
					require.Equal(t, rollout.PromotionRolloutRow, fetched.PromotionRolloutRow)
					require.Zero(t, countRows(t, rollout))
					// Only the rollout's creation has been announced.
					var announced int
					for _, e := range forProject(t, events, project) {
						if e.event.Key.Name == rollout.Name {
							announced++
						}
					}
					require.Equal(t, 1, announced)
				}

				_, err := store.Start(ctx, project, database.PromotionRolloutRow{Name: "missing"}, ref, nil)
				require.ErrorIs(t, err, database.ErrNotFound)

				// Not the Freight the rollout promotes.
				_, err = store.Start(ctx, project, rollout.PromotionRolloutRow, freight("images-2", images), nil)
				require.ErrorIs(t, err, database.ErrInvalid)
				require.ErrorContains(t, err, `promotes Freight "images-1", not "images-2"`)
				requirePending(t, rollout)

				// Freight that names no origin.
				noOrigin := ref
				noOrigin.Origin = kargoapi.FreightOrigin{}
				_, err = store.Start(ctx, project, rollout.PromotionRolloutRow, noOrigin, nil)
				require.ErrorIs(t, err, database.ErrInvalid)
				require.ErrorContains(t, err, "has no origin")
				requirePending(t, rollout)

				// Preconditions the row does not meet.
				stale := rollout.PromotionRolloutRow
				stale.UpdatedAt = stale.UpdatedAt.Add(-time.Second)
				_, err = store.Start(ctx, project, stale, ref, nil)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "modified")
				replaced := rollout.PromotionRolloutRow
				replaced.ID = uuid.New()
				_, err = store.Start(ctx, project, replaced, ref, nil)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "recreated")
				requirePending(t, rollout)

				// Another rollout of the Stage is Running.
				running := create(t, project, "prod", "prod.01B", freight("config-1", config))
				_, err = store.Start(ctx, project, running.PromotionRolloutRow, freight("config-1", config), nil)
				require.NoError(t, err)
				_, err = store.Start(ctx, project, rollout.PromotionRolloutRow, ref, nil)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "another PromotionRollout of the Stage is Running")
				requirePending(t, rollout)

				// Once that one finishes, the start goes through. The rollout
				// that ran meanwhile is numbered above this one, so it is not
				// the one before it and nothing is carried forward from it.
				finish(t, running, succeeded)
				_, err = store.Start(
					ctx, project, rollout.PromotionRolloutRow, ref, []kargoapi.FreightOrigin{images, config},
				)
				require.NoError(t, err)
				requireCollection(t, rollout, ref)
			},
		},
		{
			name: "the database rejects bad rows",
			run: func(t *testing.T) {
				const project = "bad"
				seedStage(t, project, "prod")
				rollout := create(t, project, "prod", "prod.01A", freight("images-1", images))
				insert := func(id uuid.UUID, kind, name, freightName, reference string) error {
					_, err := pool.Exec(ctx,
						`INSERT INTO promotion_rollout_freight
						 (promotion_rollout_id, origin_kind, origin_name, freight_name, reference)
						 VALUES ($1, $2, $3, $4, $5::json)`,
						id, kind, name, freightName, reference,
					)
					return err
				}
				databasetest.RequirePGError(t, insert(rollout.ID, "", "images", "f", "{}"), "23514")
				databasetest.RequirePGError(t, insert(rollout.ID, "Warehouse", "", "f", "{}"), "23514")
				databasetest.RequirePGError(t, insert(rollout.ID, "Warehouse", "images", "", "{}"), "23514")
				databasetest.RequirePGError(t, insert(rollout.ID, "Warehouse", "images", "f", "[]"), "23514")
				databasetest.RequirePGError(t, insert(rollout.ID, "Warehouse", "images", "f", "null"), "23514")
				// The value is well formed JSON.
				databasetest.RequirePGError(t, insert(rollout.ID, "Warehouse", "images", "f", "{"), "22P02")
				// A row belongs to a rollout that exists, and an origin has
				// one row per rollout.
				databasetest.RequirePGError(t, insert(uuid.New(), "Warehouse", "images", "f", "{}"), "23503")
				require.NoError(t, insert(rollout.ID, "Warehouse", "images", "f", "{}"))
				databasetest.RequirePGError(t, insert(rollout.ID, "Warehouse", "images", "g", "{}"), "23505")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.run)
	}
}

// freightNames returns the names of the collection's Freight in the order
// the collection gives its references.
func freightNames(collection *kargoapi.FreightCollection) []string {
	refs := collection.References()
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.Name
	}
	return names
}
