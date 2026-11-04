//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestTargetStagePromotionsIntegration(t *testing.T) {
	t.Parallel()
	pool, migrations := databasetest.IsolatedDatabase(t)
	queries := New(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	// Three Promotions from dev to cluster-a, created an hour apart.
	promotion := func(id string, name string, createdAt time.Time) UpsertPromotionParams {
		return UpsertPromotionParams{
			ID: id, ProjectName: "demo", Name: name, StageName: "dev", FreightName: "def456",
			TargetName: new("cluster-a"), Phase: "Running", CreatedAt: createdAt,
		}
	}
	first := promotion("id-1", "dev.1.def456", created)
	second := promotion("id-2", "dev.2.def456", created.Add(time.Hour))
	third := promotion("id-3", "dev.3.def456", created.Add(2*time.Hour))
	finished := func(params UpsertPromotionParams, phase string, at time.Time) UpsertPromotionParams {
		params.Phase = phase
		params.FinishedAt = &at
		return params
	}
	pairs := func(t *testing.T, target *string) []TargetStagePromotionRow {
		t.Helper()
		rows, err := queries.ListTargetStagePromotions(ctx, ListTargetStagePromotionsParams{
			ProjectName: "demo", TargetName: target,
		})
		require.NoError(t, err)
		return rows
	}
	// pair returns the one pair the Project has.
	pair := func(t *testing.T) TargetStagePromotionRow {
		t.Helper()
		rows := pairs(t, nil)
		require.Len(t, rows, 1)
		return rows[0]
	}
	requireLatest := func(t *testing.T, row TargetStagePromotionRow, params UpsertPromotionParams) {
		t.Helper()
		require.Equal(t, params.ID, row.LatestPromotionID)
		require.Equal(t, params.Phase, row.LatestPhase)
		require.True(t, row.LatestCreatedAt.Equal(params.CreatedAt))
	}
	requireSucceeded := func(t *testing.T, row TargetStagePromotionRow, params *UpsertPromotionParams) {
		t.Helper()
		if params == nil {
			require.Nil(t, row.SucceededPromotionID)
			require.Nil(t, row.SucceededFinishedAt)
			return
		}
		require.NotNil(t, row.SucceededPromotionID)
		require.Equal(t, params.ID, *row.SucceededPromotionID)
		require.NotNil(t, row.SucceededFinishedAt)
		require.True(t, row.SucceededFinishedAt.Equal(*params.FinishedAt))
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "a Promotion to no Target records no pair",
			run: func(t *testing.T) {
				untargeted := first
				untargeted.TargetName = nil
				requireUpsertPromotion(t, queries, untargeted)
				require.Empty(t, pairs(t, nil))
			},
		},
		{
			name: "the first Promotion to a Target is the pair's latest, and its successful one once it succeeds",
			run: func(t *testing.T) {
				requireUpsertPromotion(t, queries, first)
				row := pair(t)
				require.Equal(t, "demo", row.ProjectName)
				require.Equal(t, "cluster-a", row.TargetName)
				require.Equal(t, "dev", row.StageName)
				requireLatest(t, row, first)
				requireSucceeded(t, row, nil)
				succeeded := finished(first, "Succeeded", created.Add(time.Minute))
				requireUpsertPromotion(t, queries, succeeded)
				row = pair(t)
				requireLatest(t, row, succeeded)
				requireSucceeded(t, row, &succeeded)
			},
		},
		{
			name: "a later Promotion replaces the latest, and the successful one only once it succeeds",
			run: func(t *testing.T) {
				succeeded := finished(first, "Succeeded", created.Add(time.Minute))
				requireUpsertPromotion(t, queries, succeeded)
				requireUpsertPromotion(t, queries, second)
				row := pair(t)
				requireLatest(t, row, second)
				requireSucceeded(t, row, &succeeded)
				failed := finished(second, "Failed", second.CreatedAt.Add(time.Minute))
				requireUpsertPromotion(t, queries, failed)
				row = pair(t)
				requireLatest(t, row, failed)
				requireSucceeded(t, row, &succeeded)
				next := finished(third, "Succeeded", third.CreatedAt.Add(time.Minute))
				requireUpsertPromotion(t, queries, next)
				row = pair(t)
				requireLatest(t, row, next)
				requireSucceeded(t, row, &next)
			},
		},
		{
			// A resync can offer an earlier Promotion after a later one.
			name: "a Promotion first seen late settles by its own timestamps, not its arrival",
			run: func(t *testing.T) {
				later := finished(second, "Succeeded", second.CreatedAt.Add(time.Minute))
				requireUpsertPromotion(t, queries, later)
				earlier := finished(first, "Succeeded", created.Add(time.Minute))
				requireUpsertPromotion(t, queries, earlier)
				row := pair(t)
				requireLatest(t, row, later)
				requireSucceeded(t, row, &later)
				// Offered again, it changes nothing.
				requireUpsertPromotion(t, queries, earlier)
				require.Equal(t, row, pair(t))
			},
		},
		{
			name: "the successful Promotion is the latest to finish, whichever was created first",
			run: func(t *testing.T) {
				requireUpsertPromotion(t, queries, finished(second, "Succeeded", second.CreatedAt.Add(time.Minute)))
				slow := finished(first, "Succeeded", third.CreatedAt)
				requireUpsertPromotion(t, queries, slow)
				row := pair(t)
				require.Equal(t, second.ID, row.LatestPromotionID)
				requireSucceeded(t, row, &slow)
			},
		},
		{
			name: "a Promotion deleted unfinished gives the pair its phase only while it is the latest",
			run: func(t *testing.T) {
				requireUpsertPromotion(t, queries, first)
				requireUpsertPromotion(t, queries, second)
				finish := func(params UpsertPromotionParams) int64 {
					n, err := queries.FinishDeletedPromotion(ctx, FinishDeletedPromotionParams{
						Phase: "Aborted", ProjectName: params.ProjectName, Name: params.Name,
					})
					require.NoError(t, err)
					return n
				}
				require.Equal(t, int64(1), finish(first))
				row := pair(t)
				requireLatest(t, row, second)
				require.Equal(t, int64(1), finish(second))
				row = pair(t)
				require.Equal(t, second.ID, row.LatestPromotionID)
				require.Equal(t, "Aborted", row.LatestPhase)
				requireSucceeded(t, row, nil)
				// Already finished: neither row changes.
				require.Equal(t, int64(0), finish(second))
				require.Equal(t, row, pair(t))
			},
		},
		{
			name: "a Promotion under a taken name leaves the pair alone",
			run: func(t *testing.T) {
				succeeded := finished(first, "Succeeded", created.Add(time.Minute))
				requireUpsertPromotion(t, queries, succeeded)
				impostor := finished(first, "Failed", third.CreatedAt)
				impostor.ID = "id-impostor"
				impostor.CreatedAt = second.CreatedAt
				taken, err := queries.UpsertPromotion(ctx, impostor)
				require.NoError(t, err)
				require.True(t, taken)
				row := pair(t)
				requireLatest(t, row, succeeded)
				requireSucceeded(t, row, &succeeded)
			},
		},
		{
			name: "pairs are listed by Project, or by one Target, in Target and Stage order",
			run: func(t *testing.T) {
				otherTarget := promotion("id-b", "dev.b.def456", created)
				otherTarget.TargetName = new("cluster-b")
				otherStage := promotion("id-s", "stage2.s.def456", created)
				otherStage.StageName = "stage2"
				otherProject := promotion("id-o", "dev.o.def456", created)
				otherProject.ProjectName = "other"
				for _, params := range []UpsertPromotionParams{otherProject, otherTarget, otherStage, first} {
					requireUpsertPromotion(t, queries, params)
				}
				rows := pairs(t, nil)
				require.Len(t, rows, 3)
				require.Equal(t, []string{"cluster-a", "cluster-a", "cluster-b"}, pairTargets(rows))
				require.Equal(t, []string{"dev", "stage2", "dev"}, pairStages(rows))
				rows = pairs(t, new("cluster-a"))
				require.Len(t, rows, 2)
				require.Equal(t, []string{"dev", "stage2"}, pairStages(rows))
			},
		},
		{
			// Last, since it rolls the table's migration back and forward.
			name: "the migration records the pairs the history already holds",
			run: func(t *testing.T) {
				_, err := migrations.Down(ctx)
				require.NoError(t, err)
				insert := func(params UpsertPromotionParams) {
					_, err = pool.Exec(
						ctx,
						`INSERT INTO promotions (
							id, project_name, name, stage_name, freight_name, target_name, phase, created_at, finished_at
						) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
						params.ID, params.ProjectName, params.Name, params.StageName, params.FreightName,
						params.TargetName, params.Phase, params.CreatedAt, params.FinishedAt,
					)
					require.NoError(t, err)
				}
				succeeded := finished(first, "Succeeded", created.Add(time.Minute))
				insert(succeeded)
				insert(finished(second, "Failed", second.CreatedAt.Add(time.Minute)))
				insert(third)
				otherTarget := promotion("id-b", "dev.b.def456", created)
				otherTarget.TargetName = new("cluster-b")
				insert(otherTarget)
				untargeted := promotion("id-u", "dev.u.def456", created)
				untargeted.TargetName = nil
				insert(untargeted)
				_, err = migrations.Up(ctx)
				require.NoError(t, err)
				rows := pairs(t, nil)
				require.Len(t, rows, 2)
				requireLatest(t, rows[0], third)
				requireSucceeded(t, rows[0], &succeeded)
				require.Equal(t, "cluster-b", rows[1].TargetName)
				requireLatest(t, rows[1], otherTarget)
				requireSucceeded(t, rows[1], nil)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE promotions, promotion_freight, target_stage_promotions")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func pairTargets(rows []TargetStagePromotionRow) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.TargetName)
	}
	return names
}

func pairStages(rows []TargetStagePromotionRow) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.StageName)
	}
	return names
}
