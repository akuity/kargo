//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

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
		ID: "opaque-promotion", ProjectName: "demo", Name: "dev.01abc.def456",
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
				require.NoError(t, queries.UpsertPromotion(ctx, running))
				_, err := pool.Exec(ctx, "UPDATE promotions SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				require.NoError(t, queries.UpsertPromotion(ctx, succeeded))
				rows, err := queries.ListPromotionsByID(ctx, []string{running.ID})
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, succeeded.ID, rows[0].ID)
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
					require.NoError(t, queries.UpsertPromotion(ctx, params))
					_, err := pool.Exec(ctx, "UPDATE promotions SET synced_at = '2000-01-01'")
					require.NoError(t, err)
					require.NoError(t, queries.UpsertPromotion(ctx, params))
					rows, err := queries.ListPromotionsByID(ctx, []string{params.ID})
					require.NoError(t, err)
					require.Len(t, rows, 1)
					require.True(t, rows[0].SyncedAt.Equal(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
				}
			},
		},
		{
			name: "an unfinished Promotion to no Target has no finish time or Target",
			run: func(t *testing.T) {
				require.NoError(t, queries.UpsertPromotion(ctx, running))
				rows, err := queries.ListPromotionsByID(ctx, []string{running.ID})
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
				require.NoError(t, queries.UpsertPromotion(ctx, targeted))
				rows, err := queries.ListPromotionsByID(ctx, []string{running.ID})
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, "cluster-a", *rows[0].TargetName)
				targeted.TargetName = new("")
				databasetest.RequirePGError(t, queries.UpsertPromotion(ctx, targeted), "23514")
			},
		},
		{
			name: "snapshot returns only the rows asked for, in ID order",
			run: func(t *testing.T) {
				b := running
				b.ID = "b"
				a := running
				a.ID = "a"
				for _, params := range []UpsertPromotionParams{running, b, a} {
					require.NoError(t, queries.UpsertPromotion(ctx, params))
				}
				rows, err := queries.ListPromotionsByID(ctx, []string{"b", "absent", "a"})
				require.NoError(t, err)
				require.Equal(t, []string{"a", "b"}, promotionIDs(rows))
				rows, err = queries.ListPromotionsByID(ctx, nil)
				require.NoError(t, err)
				require.Empty(t, rows)
			},
		},
		{
			// A Promotion recreated under a name an earlier one had is a new
			// Promotion; both rows stay.
			name: "names are not unique, identities are",
			run: func(t *testing.T) {
				require.NoError(t, queries.UpsertPromotion(ctx, succeeded))
				recreated := running
				recreated.ID = "recreated"
				require.NoError(t, queries.UpsertPromotion(ctx, recreated))
				rows, err := queries.ListPromotionsByID(ctx, []string{succeeded.ID, recreated.ID})
				require.NoError(t, err)
				require.Equal(t, []string{succeeded.ID, recreated.ID}, promotionIDs(rows))
				_, err = pool.Exec(ctx, "INSERT INTO promotions SELECT * FROM promotions")
				databasetest.RequirePGError(t, err, "23505")
			},
		},
		{
			name: "a Promotion's collection is recorded once, by origin",
			run: func(t *testing.T) {
				withCollection := running
				withCollection.OriginKinds = []string{"Warehouse", "Warehouse"}
				withCollection.OriginNames = []string{"w1", "w2"}
				withCollection.FreightNames = []string{"f1", "f2"}
				require.NoError(t, queries.UpsertPromotion(ctx, running))
				require.Empty(t, promotionFreight(t, pool, running.ID))
				// The collection arrives with nothing else changed, so the
				// Promotion's row is not updated, and the collection is
				// recorded all the same.
				require.NoError(t, queries.UpsertPromotion(ctx, withCollection))
				want := []string{"Warehouse/w1=f1", "Warehouse/w2=f2"}
				require.Equal(t, want, promotionFreight(t, pool, running.ID))
				// A collection, once recorded, is kept, whatever a later sync says.
				changed := withCollection
				changed.FreightNames = []string{"other", "other"}
				require.NoError(t, queries.UpsertPromotion(ctx, changed))
				require.NoError(t, queries.UpsertPromotion(ctx, succeeded))
				require.Equal(t, want, promotionFreight(t, pool, running.ID))
				// A Promotion first seen with its collection gets both at once.
				fresh := withCollection
				fresh.ID = "fresh"
				require.NoError(t, queries.UpsertPromotion(ctx, fresh))
				require.Equal(t, want, promotionFreight(t, pool, fresh.ID))
				empty := running
				empty.ID = "empty"
				empty.OriginKinds = []string{""}
				empty.OriginNames = []string{"w"}
				empty.FreightNames = []string{"f"}
				databasetest.RequirePGError(t, queries.UpsertPromotion(ctx, empty), "23514")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE promotions, promotion_freight")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func promotionIDs(rows []PromotionRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

// promotionFreight lists a Promotion's recorded Freight as origin=name, in
// origin order.
func promotionFreight(t *testing.T, pool *pgxpool.Pool, id string) []string {
	t.Helper()
	rows, err := pool.Query(
		context.Background(),
		`SELECT origin_kind || '/' || origin_name || '=' || freight_name
		FROM promotion_freight WHERE promotion_id = $1
		ORDER BY origin_kind, origin_name`,
		id,
	)
	require.NoError(t, err)
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return out
}
