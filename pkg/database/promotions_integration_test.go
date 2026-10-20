//go:build integration && db

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestPromotionsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := New(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := created.Add(time.Minute)
	running := UpsertPromotionParams{
		ID: "id-running", ProjectName: "demo", Name: "dev.01abc.def456",
		StageName: "dev", FreightName: "def456", Phase: "Running", CreatedAt: created,
	}
	succeeded := running
	succeeded.Phase = "Succeeded"
	succeeded.FinishedAt = &finished
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "a row follows its Promotion's phase and refreshes its sync timestamp",
			run: func(t *testing.T) {
				requireUpsertPromotion(t, queries, running)
				_, err := pool.Exec(ctx, "UPDATE promotions SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				requireUpsertPromotion(t, queries, succeeded)
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, succeeded.ProjectName, rows[0].ProjectName)
				require.Equal(t, succeeded.Name, rows[0].Name)
				require.Equal(t, succeeded.StageName, rows[0].StageName)
				require.Equal(t, succeeded.FreightName, rows[0].FreightName)
				require.Equal(t, succeeded.Phase, rows[0].Phase)
				require.True(t, rows[0].CreatedAt.Equal(created))
				require.NotNil(t, rows[0].FinishedAt)
				require.True(t, rows[0].FinishedAt.Equal(finished))
				require.True(t, rows[0].SyncedAt.After(created))
			},
		},
		{
			name: "an upsert that changes nothing leaves the row alone",
			run: func(t *testing.T) {
				for _, params := range []UpsertPromotionParams{running, succeeded} {
					requireUpsertPromotion(t, queries, params)
					_, err := pool.Exec(ctx, "UPDATE promotions SET synced_at = '2000-01-01'")
					require.NoError(t, err)
					requireUpsertPromotion(t, queries, params)
					rows, err := queries.ListPromotionsByKey(ctx, keysOf(params))
					require.NoError(t, err)
					require.Len(t, rows, 1)
					require.True(t, rows[0].SyncedAt.Equal(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
				}
			},
		},
		{
			// Every sync offers a new id; the row keeps the one it was first
			// written with, and so does a collection recorded later.
			name: "a row keeps the id it was first written with",
			run: func(t *testing.T) {
				requireUpsertPromotion(t, queries, running)
				resynced := succeeded
				resynced.ID = "id-offered-later"
				resynced.OriginKinds = []string{"Warehouse"}
				resynced.OriginNames = []string{"w1"}
				resynced.FreightNames = []string{"f1"}
				requireUpsertPromotion(t, queries, resynced)
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, running.ID, rows[0].ID)
				require.Equal(t, succeeded.Phase, rows[0].Phase)
				var freightOwner string
				require.NoError(t, pool.QueryRow(ctx, "SELECT promotion_id FROM promotion_freight").Scan(&freightOwner))
				require.Equal(t, running.ID, freightOwner)
			},
		},
		{
			name: "an unfinished Promotion to no Target has no finish time or Target",
			run: func(t *testing.T) {
				requireUpsertPromotion(t, queries, running)
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Nil(t, rows[0].FinishedAt)
				require.Nil(t, rows[0].TargetName)
			},
		},
		{
			name: "a Promotion to a Target names it, and the name cannot be empty",
			run: func(t *testing.T) {
				targeted := running
				targeted.TargetName = new("cluster-a")
				requireUpsertPromotion(t, queries, targeted)
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, "cluster-a", *rows[0].TargetName)
				targeted.TargetName = new("")
				_, err = queries.UpsertPromotion(ctx, targeted)
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "snapshot returns only the rows asked for, in key order",
			run: func(t *testing.T) {
				b := running
				b.ID = "id-b"
				b.Name = "b"
				a := running
				a.ID = "id-a"
				a.Name = "a"
				// The same name in another Project is another Promotion.
				elsewhere := running
				elsewhere.ID = "id-elsewhere"
				elsewhere.ProjectName = "other"
				elsewhere.Name = "a"
				for _, params := range []UpsertPromotionParams{running, b, a, elsewhere} {
					requireUpsertPromotion(t, queries, params)
				}
				absent := running
				absent.Name = "absent"
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(b, absent, a))
				require.NoError(t, err)
				require.Equal(t, []string{"demo/a", "demo/b"}, promotionKeys(rows))
				rows, err = queries.ListPromotionsByKey(ctx, keysOf(elsewhere, a))
				require.NoError(t, err)
				require.Equal(t, []string{"demo/a", "other/a"}, promotionKeys(rows))
				rows, err = queries.ListPromotionsByKey(ctx, keysOf())
				require.NoError(t, err)
				require.Empty(t, rows)
			},
		},
		{
			// A different Promotion under a name already recorded is reported,
			// and the record, with its collection, is left as it is.
			name: "a name already recorded for an earlier Promotion is reported and kept",
			run: func(t *testing.T) {
				recorded := succeeded
				recorded.OriginKinds = []string{"Warehouse"}
				recorded.OriginNames = []string{"w1"}
				recorded.FreightNames = []string{"f1"}
				requireUpsertPromotion(t, queries, recorded)
				later := running
				later.ID = "id-later"
				later.CreatedAt = created.Add(time.Hour)
				later.FreightName = "other"
				later.OriginKinds = []string{"Warehouse", "Warehouse"}
				later.OriginNames = []string{"w1", "w2"}
				later.FreightNames = []string{"other", "other"}
				taken, err := queries.UpsertPromotion(ctx, later)
				require.NoError(t, err)
				require.True(t, taken)
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.True(t, rows[0].CreatedAt.Equal(created))
				require.Equal(t, succeeded.FreightName, rows[0].FreightName)
				require.Equal(t, succeeded.Phase, rows[0].Phase)
				require.Equal(t, []string{"Warehouse/w1=f1"}, promotionFreight(t, pool, running))
				_, err = pool.Exec(ctx, "INSERT INTO promotions SELECT * FROM promotions")
				databasetest.RequirePGError(t, err, "23505")
			},
		},
		{
			// Only unfinished rows are listed, and only an unfinished row is
			// finished: a finished one is the record of what the Promotion did.
			name: "a Promotion deleted before it finished is recorded as finished",
			run: func(t *testing.T) {
				other := running
				other.Name = "dev.02def.abc123"
				other.ID = "id-other"
				for _, params := range []UpsertPromotionParams{running, other, {
					ID: "id-done", ProjectName: "demo", Name: "dev.00abc.def456", StageName: "dev",
					FreightName: "def456", Phase: "Succeeded", CreatedAt: created, FinishedAt: &finished,
				}} {
					requireUpsertPromotion(t, queries, params)
				}
				unfinished, err := queries.ListUnfinishedPromotions(ctx)
				require.NoError(t, err)
				require.Equal(t, []string{"demo/" + running.Name, "demo/" + other.Name}, promotionKeys(unfinished))
				finish := FinishDeletedPromotionParams{Phase: "Aborted", ProjectName: "demo", Name: running.Name}
				n, err := queries.FinishDeletedPromotion(ctx, finish)
				require.NoError(t, err)
				require.EqualValues(t, 1, n)
				rows, err := queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, "Aborted", rows[0].Phase)
				require.NotNil(t, rows[0].FinishedAt)
				require.True(t, rows[0].FinishedAt.After(created))
				// Finished once, the row is left alone, whatever phase is asked.
				finish.Phase = "Failed"
				n, err = queries.FinishDeletedPromotion(ctx, finish)
				require.NoError(t, err)
				require.Zero(t, n)
				rows, err = queries.ListPromotionsByKey(ctx, keysOf(running))
				require.NoError(t, err)
				require.Equal(t, "Aborted", rows[0].Phase)
				n, err = queries.FinishDeletedPromotion(ctx, FinishDeletedPromotionParams{
					Phase: "Aborted", ProjectName: "demo", Name: "dev.00abc.def456",
				})
				require.NoError(t, err)
				require.Zero(t, n)
				unfinished, err = queries.ListUnfinishedPromotions(ctx)
				require.NoError(t, err)
				require.Equal(t, []string{"demo/" + other.Name}, promotionKeys(unfinished))
			},
		},
		{
			name: "a Promotion's collection is recorded once, by origin",
			run: func(t *testing.T) {
				withCollection := running
				withCollection.OriginKinds = []string{"Warehouse", "Warehouse"}
				withCollection.OriginNames = []string{"w1", "w2"}
				withCollection.FreightNames = []string{"f1", "f2"}
				requireUpsertPromotion(t, queries, running)
				require.Empty(t, promotionFreight(t, pool, running))
				// The collection arrives with nothing else changed, so the
				// Promotion's row is not updated, and the collection is
				// recorded all the same.
				requireUpsertPromotion(t, queries, withCollection)
				want := []string{"Warehouse/w1=f1", "Warehouse/w2=f2"}
				require.Equal(t, want, promotionFreight(t, pool, running))
				// A collection, once recorded, is kept, whatever a later sync says.
				changed := withCollection
				changed.FreightNames = []string{"other", "other"}
				requireUpsertPromotion(t, queries, changed)
				requireUpsertPromotion(t, queries, succeeded)
				require.Equal(t, want, promotionFreight(t, pool, running))
				// A Promotion first seen with its collection gets both at once.
				fresh := withCollection
				fresh.ID = "id-fresh"
				fresh.Name = "fresh"
				requireUpsertPromotion(t, queries, fresh)
				require.Equal(t, want, promotionFreight(t, pool, fresh))
				empty := running
				empty.ID = "id-empty"
				empty.Name = "empty"
				empty.OriginKinds = []string{""}
				empty.OriginNames = []string{"w"}
				empty.FreightNames = []string{"f"}
				_, err := queries.UpsertPromotion(ctx, empty)
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "current Freight is that of each Stage's latest successful Promotion to each Target",
			run: func(t *testing.T) {
				a := seedTarget(t, queries, "cluster-a")
				b := seedTarget(t, queries, "cluster-b")
				for _, stage := range []string{"dev", "prod"} {
					require.NoError(t, queries.UpsertStage(ctx, UpsertStageParams{
						ID: stage, ProjectName: "demo", Name: stage, CreatedAt: created,
					}))
				}
				at := func(minutes int) *time.Time {
					when := created.Add(time.Duration(minutes) * time.Minute)
					return &when
				}
				promote := func(id, stage, target, phase string, finishedAt *time.Time, freight ...string) {
					params := UpsertPromotionParams{
						ID: id, ProjectName: "demo", Name: id, StageName: stage,
						FreightName: freight[0], TargetName: &target, Phase: phase,
						CreatedAt: created, FinishedAt: finishedAt,
					}
					for i, name := range freight {
						params.OriginKinds = append(params.OriginKinds, "Warehouse")
						params.OriginNames = append(params.OriginNames, fmt.Sprintf("w%d", i+1))
						params.FreightNames = append(params.FreightNames, name)
					}
					requireUpsertPromotion(t, queries, params)
				}
				promote("dev-a-old", "dev", "cluster-a", "Succeeded", at(1), "old")
				promote("dev-a-new", "dev", "cluster-a", "Succeeded", at(2), "new1", "new2")
				promote("dev-a-failed", "dev", "cluster-a", "Failed", at(3), "failed")
				promote("dev-a-running", "dev", "cluster-a", "Running", nil, "running")
				promote("prod-a", "prod", "cluster-a", "Succeeded", at(1), "prod")
				promote("dev-b", "dev", "cluster-b", "Succeeded", at(1), "b")
				// Promotions from a Stage the database does not know say nothing.
				promote("gone-a", "gone", "cluster-a", "Succeeded", at(9), "gone")

				type entry struct {
					target  uuid.UUID
					stage   string
					freight string
				}
				entries := func(rows []ListTargetCurrentFreightRow) []entry {
					out := make([]entry, 0, len(rows))
					for _, row := range rows {
						out = append(out, entry{
							row.TargetID, row.StageName,
							row.OriginKind + "/" + row.OriginName + "=" + row.FreightName,
						})
					}
					return out
				}
				wantA := []entry{
					{a.ID, "dev", "Warehouse/w1=new1"},
					{a.ID, "dev", "Warehouse/w2=new2"},
					{a.ID, "prod", "Warehouse/w1=prod"},
				}
				wantB := []entry{{b.ID, "dev", "Warehouse/w1=b"}}
				all := entries(currentFreight(t, queries, ""))
				require.ElementsMatch(t, append(wantA, wantB...), all)
				require.Equal(t, wantA, entries(currentFreight(t, queries, "cluster-a")))
				other, err := queries.ListTargetCurrentFreight(
					ctx, ListTargetCurrentFreightParams{ProjectName: "other"},
				)
				require.NoError(t, err)
				require.Empty(t, other)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE promotions, promotion_freight, targets, stages CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

// requireUpsertPromotion upserts a Promotion whose name no other Promotion was
// recorded under.
func requireUpsertPromotion(t *testing.T, queries *Queries, params UpsertPromotionParams) {
	t.Helper()
	taken, err := queries.UpsertPromotion(context.Background(), params)
	require.NoError(t, err)
	require.False(t, taken)
}

// keysOf returns the snapshot parameters that ask for the given Promotions.
func keysOf(promotions ...UpsertPromotionParams) ListPromotionsByKeyParams {
	keys := ListPromotionsByKeyParams{ProjectNames: []string{}, Names: []string{}}
	for _, promotion := range promotions {
		keys.ProjectNames = append(keys.ProjectNames, promotion.ProjectName)
		keys.Names = append(keys.Names, promotion.Name)
	}
	return keys
}

func promotionKeys(rows []PromotionRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.ProjectName+"/"+row.Name)
	}
	return keys
}

func seedTarget(t *testing.T, queries *Queries, name string) TargetRow {
	t.Helper()
	target, err := queries.CreateTarget(context.Background(), CreateTargetParams{
		ProjectName:   "demo",
		Name:          name,
		Labels:        []byte(`{}`),
		Params:        []byte(`{}`),
		TargetSetName: "fleet",
	})
	require.NoError(t, err)
	return target
}

// currentFreight lists the demo Project's Targets' current Freight, or only
// the named Target's.
func currentFreight(t *testing.T, queries *Queries, target string) []ListTargetCurrentFreightRow {
	t.Helper()
	args := ListTargetCurrentFreightParams{ProjectName: "demo"}
	if target != "" {
		args.TargetName = &target
	}
	rows, err := queries.ListTargetCurrentFreight(context.Background(), args)
	require.NoError(t, err)
	return rows
}

// promotionFreight lists a Promotion's recorded Freight as origin=name, in
// origin order.
func promotionFreight(t *testing.T, pool *pgxpool.Pool, promotion UpsertPromotionParams) []string {
	t.Helper()
	rows, err := pool.Query(
		context.Background(),
		`SELECT origin_kind || '/' || origin_name || '=' || promotion_freight.freight_name
		FROM promotion_freight JOIN promotions ON promotions.id = promotion_freight.promotion_id
		WHERE promotions.project_name = $1 AND promotions.name = $2
		ORDER BY origin_kind, origin_name`,
		promotion.ProjectName,
		promotion.Name,
	)
	require.NoError(t, err)
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return out
}
