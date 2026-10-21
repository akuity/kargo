//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

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
			name: "recreation replaces the row",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				replacement := stage
				replacement.ID = "new-stage"
				require.NoError(t, replace(replacement))
				ids, err := listStageIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{replacement.ID}, ids)
			},
		},
		{
			name: "deletion by Project and name",
			run: func(t *testing.T) {
				require.NoError(t, replace(stage))
				other := stage
				other.ID = "other-stage"
				other.ProjectName = "other"
				require.NoError(t, replace(other))
				require.NoError(t, queries.DeleteStageByName(ctx, DeleteStageByNameParams{
					ProjectName: "other", Name: "dev",
				}))
				ids, err := listStageIDs(ctx, queries)
				require.NoError(t, err)
				require.Equal(t, []string{stage.ID}, ids)
				require.NoError(t, queries.DeleteStageByName(ctx, DeleteStageByNameParams{
					ProjectName: "demo", Name: "absent",
				}))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE stages")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
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
