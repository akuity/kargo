//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestStagesIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := New(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	stage := UpsertStageParams{ID: "opaque-stage", ProjectName: "demo", Name: "dev", CreatedAt: created}
	// replace writes a Stage the way the mirror does.
	replace := func(params UpsertStageParams) error {
		if err := queries.DeleteReplacedStage(ctx, DeleteReplacedStageParams{
			ProjectName: params.ProjectName,
			Name:        params.Name,
			ID:          params.ID,
		}); err != nil {
			return err
		}
		return queries.UpsertStage(ctx, params)
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "a write that changes nothing leaves the row alone",
			run: func(t *testing.T) {
				fleet := stage
				fleet.TargetSelectors = []byte(`[{"matchLabels":{"region":"us"}}]`)
				for _, params := range []UpsertStageParams{stage, fleet} {
					require.NoError(t, replace(params))
					_, err := pool.Exec(ctx, "UPDATE stages SET synced_at = '2000-01-01'")
					require.NoError(t, err)
					// The same selectors in other spacing are the same JSONB.
					same := params
					if same.TargetSelectors != nil {
						same.TargetSelectors = []byte(`[ { "matchLabels": { "region": "us" } } ]`)
					}
					require.NoError(t, replace(same))
					ids, err := listStageIDs(ctx, queries)
					require.NoError(t, err)
					require.Equal(t, []string{stage.ID}, ids)
					var creation, synced time.Time
					require.NoError(t, pool.QueryRow(ctx, "SELECT created_at, synced_at FROM stages").Scan(&creation, &synced))
					require.True(t, creation.Equal(created))
					require.True(t, synced.Equal(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
				}
			},
		},
		{
			name: "a write that changes the row refreshes its sync timestamp",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				_, err := pool.Exec(ctx, "UPDATE stages SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				fleet := stage
				fleet.TargetSelectors = []byte(`[]`)
				require.NoError(t, replace(fleet))
				var synced time.Time
				require.NoError(t, pool.QueryRow(ctx, "SELECT synced_at FROM stages").Scan(&synced))
				require.True(t, synced.After(created))
			},
		},
		{
			name: "snapshot query includes all mirrored fields",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				stages, err := queries.ListStages(ctx)
				require.NoError(t, err)
				require.Len(t, stages, 1)
				require.Equal(t, stage.ID, stages[0].ID)
				require.Equal(t, stage.ProjectName, stages[0].ProjectName)
				require.Equal(t, stage.Name, stages[0].Name)
				require.True(t, stages[0].CreatedAt.Equal(created))
				require.False(t, stages[0].SyncedAt.IsZero())
				require.Nil(t, stages[0].TargetSelectors)
				require.Zero(t, stages[0].LastPromotionRolloutNumber)
			},
		},
		{
			name: "selectors follow the Stage and the rollout counter is left alone",
			run: func(t *testing.T) {
				fleet := stage
				fleet.TargetSelectors = []byte(`[{"matchLabels":{"region":"us"}}]`)
				require.NoError(t, replace(fleet))
				_, err := pool.Exec(ctx, "UPDATE stages SET last_promotion_rollout_number = 7")
				require.NoError(t, err)
				fleet.TargetSelectors = []byte(`[]`)
				require.NoError(t, replace(fleet))
				stages, err := queries.ListStages(ctx)
				require.NoError(t, err)
				require.Len(t, stages, 1)
				require.JSONEq(t, `[]`, string(stages[0].TargetSelectors))
				require.EqualValues(t, 7, stages[0].LastPromotionRolloutNumber)
				// A Stage that became classic has no selectors at all.
				require.NoError(t, replace(stage))
				stages, err = queries.ListStages(ctx)
				require.NoError(t, err)
				require.Nil(t, stages[0].TargetSelectors)
				require.EqualValues(t, 7, stages[0].LastPromotionRolloutNumber)
			},
		},
		{
			name: "selectors must be a list and the counter cannot be negative",
			run: func(t *testing.T) {
				notAList := stage
				notAList.TargetSelectors = []byte(`{"matchLabels":{}}`)
				databasetest.RequirePGError(t, replace(notAList), "23514")
				require.NoError(t, replace(stage))
				_, err := pool.Exec(ctx, "UPDATE stages SET last_promotion_rollout_number = -1")
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "the same name in another Project is another Stage",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				other := stage
				other.ID = "other-stage"
				other.ProjectName = "other"
				require.NoError(t, replace(other))
				ids, err := listStageIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{stage.ID, other.ID}, ids)
			},
		},
		{
			name: "database enforces unique names within a Project and unique identities",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				duplicate := stage
				duplicate.ID = "another"
				databasetest.RequirePGError(t, queries.UpsertStage(ctx, duplicate), "23505")
				_, err := pool.Exec(ctx, "INSERT INTO stages SELECT * FROM stages")
				databasetest.RequirePGError(t, err, "23505")
			},
		},
		{
			name: "recreation replaces the row and the old Stage's history goes with it",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				requireStageHistory(t, queries, "demo", "dev")
				replacement := stage
				replacement.ID = "new-stage"
				require.NoError(t, replace(replacement))
				ids, err := listStageIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{replacement.ID}, ids)
				require.Equal(t, stageHistory{}, countStageHistory(t, pool))
			},
		},
		{
			name: "deletion by Project and name takes the Stage's history with it",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				requireStageHistory(t, queries, "demo", "dev")
				other := stage
				other.ID = "other-stage"
				other.ProjectName = "other"
				require.NoError(t, replace(other))
				requireStageHistory(t, queries, "other", "dev")
				require.Equal(t, stageHistory{
					Rollouts: 2, RolloutTargets: 2, Promotions: 2, Freight: 2,
				}, countStageHistory(t, pool))
				require.NoError(t, queries.DeleteStageByName(ctx, DeleteStageByNameParams{
					ProjectName: "other", Name: "dev",
				}))
				ids, err := listStageIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{stage.ID}, ids)
				// The other Project's Stage of the same name keeps its own.
				require.Equal(t, stageHistory{
					Rollouts: 1, RolloutTargets: 1, Promotions: 1, Freight: 1,
				}, countStageHistory(t, pool))
				require.NoError(t, queries.DeleteStageByName(ctx, DeleteStageByNameParams{
					ProjectName: "demo", Name: "absent",
				}))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE stages, targets CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

// stageHistory counts the rows that hang off Stages: rollouts and their
// Target snapshots, Promotions and their Freight.
type stageHistory struct {
	Rollouts       int
	RolloutTargets int
	Promotions     int
	Freight        int
}

// requireStageHistory records a rollout of the Stage to one Target, and a
// Promotion of the Stage by that rollout, with one piece of Freight.
func requireStageHistory(t *testing.T, queries *Queries, project, stage string) {
	t.Helper()
	ctx := context.Background()
	target, err := queries.CreateTarget(ctx, CreateTargetParams{
		ProjectName:   project,
		Name:          "target",
		Labels:        []byte("{}"),
		Params:        []byte("{}"),
		TargetSetName: "fleet",
	})
	require.NoError(t, err)
	rollout, err := queries.CreatePromotionRollout(ctx, CreatePromotionRolloutParams{
		ProjectName: project,
		StageName:   stage,
		Name:        stage + ".rollout",
		FreightName: "def456",
		TargetIds:   []uuid.UUID{target.ID},
	})
	require.NoError(t, err)
	targetName := "target"
	requireUpsertPromotion(t, queries, UpsertPromotionParams{
		ID:                 project + "-" + stage,
		ProjectName:        project,
		Name:               stage + ".promotion",
		StageName:          stage,
		FreightName:        "def456",
		TargetName:         &targetName,
		PromotionRolloutID: &rollout.ID,
		Phase:              "Succeeded",
		CreatedAt:          time.Now().Truncate(time.Microsecond),
		OriginKinds:        []string{"Warehouse"},
		OriginNames:        []string{"w"},
		FreightNames:       []string{"def456"},
	})
}

func countStageHistory(t *testing.T, pool *pgxpool.Pool) stageHistory {
	t.Helper()
	var counts stageHistory
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM promotion_rollouts),
		(SELECT count(*) FROM promotion_rollout_targets),
		(SELECT count(*) FROM promotions),
		(SELECT count(*) FROM promotion_freight)`,
	).Scan(&counts.Rollouts, &counts.RolloutTargets, &counts.Promotions, &counts.Freight))
	return counts
}

func listStageIDs(ctx context.Context, queries *Queries) ([]string, error) {
	rows, err := queries.ListStages(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}
