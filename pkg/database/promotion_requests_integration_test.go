//go:build integration && db

package database

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestPromotionRequestsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := New(pool)
	ctx := context.Background()
	newRequest := func(name string) CreatePromotionRequestParams {
		return CreatePromotionRequestParams{
			ProjectName:    "demo",
			Name:           name,
			StageName:      "prod",
			FreightName:    "abc123",
			UpdateStrategy: []byte(`{"maxConcurrent": 2}`),
			CreatedBy:      "alice",
		}
	}
	create := func(t *testing.T, name string) PromotionRequestRow {
		row, err := queries.CreatePromotionRequest(ctx, newRequest(name))
		require.NoError(t, err)
		return row
	}
	addTargets := func(t *testing.T, id uuid.UUID, names ...string) {
		for _, name := range names {
			_, err := queries.AddPromotionRequestTarget(ctx, AddPromotionRequestTargetParams{
				PromotionRequestID: id,
				TargetName:         name,
			})
			require.NoError(t, err)
		}
	}
	createTarget := func(t *testing.T, name string) {
		_, err := queries.CreateTarget(ctx, CreateTargetParams{
			ProjectName: "demo",
			Name:        name,
			Labels:      []byte(`{}`),
			Params:      []byte(`{}`),
		})
		require.NoError(t, err)
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create returns the stored request",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				require.NotEqual(t, uuid.Nil, row.ID)
				require.Equal(t, "demo", row.ProjectName)
				require.Equal(t, "prod", row.StageName)
				require.Equal(t, "abc123", row.FreightName)
				require.Equal(t, "alice", row.CreatedBy)
				require.Equal(t, int64(1), row.Number)
				require.Equal(t, "Pending", row.Phase)
				require.Empty(t, row.Message)
				require.JSONEq(t, `{"maxConcurrent": 2}`, string(row.UpdateStrategy))
				require.True(t, row.CreatedAt.Equal(row.UpdatedAt))
				require.Nil(t, row.StartedAt)
				require.Nil(t, row.FinishedAt)
				fetched, err := queries.GetPromotionRequest(ctx, GetPromotionRequestParams{
					ProjectName: "demo",
					Name:        "req-1",
				})
				require.NoError(t, err)
				require.Equal(t, row, fetched)
				byID, err := queries.GetPromotionRequestByID(ctx, row.ID)
				require.NoError(t, err)
				require.Equal(t, row, byID)
			},
		},
		{
			name: "names are unique within a Project only",
			run: func(t *testing.T) {
				create(t, "req-1")
				_, err := queries.CreatePromotionRequest(ctx, newRequest("req-1"))
				databasetest.RequirePGError(t, err, "23505")
				// The failed create took no number with it.
				require.Equal(t, int64(2), create(t, "req-2").Number)
				other := newRequest("req-1")
				other.ProjectName = "other"
				_, err = queries.CreatePromotionRequest(ctx, other)
				require.NoError(t, err)
			},
		},
		{
			name: "concurrent creates for a Stage take distinct numbers without gaps",
			run: func(t *testing.T) {
				const creates = 10
				numbers := make([]int64, creates)
				errs := make([]error, creates)
				var wg sync.WaitGroup
				for i := range creates {
					wg.Add(1)
					go func() {
						defer wg.Done()
						row, err := queries.CreatePromotionRequest(
							ctx,
							newRequest(fmt.Sprintf("req-%d", i)),
						)
						numbers[i], errs[i] = row.Number, err
					}()
				}
				wg.Wait()
				for _, err := range errs {
					require.NoError(t, err)
				}
				slices.Sort(numbers)
				for i, number := range numbers {
					require.Equal(t, int64(i+1), number)
				}
			},
		},
		{
			name: "the database rejects unknown phases and a non-object strategy",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				_, err := queries.UpdatePromotionRequestStatus(ctx, UpdatePromotionRequestStatusParams{
					ID:    row.ID,
					Phase: "Done",
				})
				databasetest.RequirePGError(t, err, "23514")
				bad := newRequest("req-2")
				bad.UpdateStrategy = []byte(`[]`)
				_, err = queries.CreatePromotionRequest(ctx, bad)
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "lists follow creation order, by Project and by Stage",
			run: func(t *testing.T) {
				first := create(t, "req-b")
				second := create(t, "req-a")
				staging := newRequest("req-c")
				staging.StageName = "staging"
				third, err := queries.CreatePromotionRequest(ctx, staging)
				require.NoError(t, err)
				// Each Stage numbers its own requests from 1.
				require.Equal(t, int64(1), first.Number)
				require.Equal(t, int64(2), second.Number)
				require.Equal(t, int64(1), third.Number)
				rows, err := queries.ListPromotionRequests(ctx, "demo")
				require.NoError(t, err)
				require.Equal(t, []string{"req-b", "req-a", "req-c"}, requestNames(rows))
				rows, err = queries.ListPromotionRequestsByStage(ctx, ListPromotionRequestsByStageParams{
					ProjectName: "demo",
					StageName:   "prod",
				})
				require.NoError(t, err)
				require.Equal(t, []string{"req-b", "req-a"}, requestNames(rows))
				rows, err = queries.ListPromotionRequests(ctx, "missing")
				require.NoError(t, err)
				require.Empty(t, rows)
			},
		},
		{
			name: "open requests are paged by id",
			run: func(t *testing.T) {
				open := create(t, "req-1")
				closed := create(t, "req-2")
				finished := time.Now()
				_, err := queries.UpdatePromotionRequestStatus(ctx, UpdatePromotionRequestStatusParams{
					ID:         closed.ID,
					Phase:      "Succeeded",
					FinishedAt: &finished,
				})
				require.NoError(t, err)
				ids, err := queries.ListOpenPromotionRequestIDs(ctx, ListOpenPromotionRequestIDsParams{
					AfterID:  uuid.Nil,
					RowLimit: 10,
				})
				require.NoError(t, err)
				require.Equal(t, []uuid.UUID{open.ID}, ids)
				ids, err = queries.ListOpenPromotionRequestIDs(ctx, ListOpenPromotionRequestIDsParams{
					AfterID:  open.ID,
					RowLimit: 10,
				})
				require.NoError(t, err)
				require.Empty(t, ids)
			},
		},
		{
			name: "status updates advance updated_at even when the clock does not",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				_, err := pool.Exec(ctx, "UPDATE promotion_requests SET updated_at = '2999-01-01'")
				require.NoError(t, err)
				started := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
				updated, err := queries.UpdatePromotionRequestStatus(ctx, UpdatePromotionRequestStatusParams{
					ID:        row.ID,
					Phase:     "Running",
					Message:   "1 of 2 Targets",
					StartedAt: &started,
				})
				require.NoError(t, err)
				require.Equal(t, "Running", updated.Phase)
				require.Equal(t, "1 of 2 Targets", updated.Message)
				require.True(t, updated.StartedAt.Equal(started))
				require.Nil(t, updated.FinishedAt)
				require.True(t, updated.UpdatedAt.After(time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC)))
				_, err = queries.UpdatePromotionRequestStatus(ctx, UpdatePromotionRequestStatusParams{
					ID:    uuid.New(),
					Phase: "Running",
				})
				require.ErrorIs(t, err, pgx.ErrNoRows)
			},
		},
		{
			name: "Targets are added once each and listed by name",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				addTargets(t, row.ID, "us-east", "eu-west")
				_, err := queries.AddPromotionRequestTarget(ctx, AddPromotionRequestTargetParams{
					PromotionRequestID: row.ID,
					TargetName:         "us-east",
				})
				require.ErrorIs(t, err, pgx.ErrNoRows)
				targets, err := queries.ListPromotionRequestTargets(ctx, []uuid.UUID{row.ID})
				require.NoError(t, err)
				require.Equal(t, []PromotionRequestTargetRow{
					{PromotionRequestID: row.ID, TargetName: "eu-west"},
					{PromotionRequestID: row.ID, TargetName: "us-east"},
				}, targets)
				_, err = queries.AddPromotionRequestTarget(ctx, AddPromotionRequestTargetParams{
					PromotionRequestID: uuid.New(),
					TargetName:         "us-east",
				})
				databasetest.RequirePGError(t, err, "23503")
			},
		},
		{
			name: "a Target's outcome is recorded against the request",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				addTargets(t, row.ID, "us-east")
				params := UpdatePromotionRequestTargetParams{
					PromotionRequestID: row.ID,
					TargetName:         "us-east",
					Promotion:          "prod.01abc.us-east",
					Phase:              "Running",
				}
				updated, err := queries.UpdatePromotionRequestTarget(ctx, params)
				require.NoError(t, err)
				require.EqualValues(t, 1, updated)
				targets, err := queries.ListPromotionRequestTargets(ctx, []uuid.UUID{row.ID})
				require.NoError(t, err)
				require.Equal(t, "prod.01abc.us-east", targets[0].Promotion)
				require.Equal(t, "Running", targets[0].Phase)
				params.Phase = "Done"
				_, err = queries.UpdatePromotionRequestTarget(ctx, params)
				databasetest.RequirePGError(t, err, "23514")
				params.TargetName = "absent"
				params.Phase = "Running"
				updated, err = queries.UpdatePromotionRequestTarget(ctx, params)
				require.NoError(t, err)
				require.Zero(t, updated)
			},
		},
		{
			name: "requests link to Targets by name in both directions",
			run: func(t *testing.T) {
				createTarget(t, "us-east")
				createTarget(t, "eu-west")
				createTarget(t, "ap-south")
				first := create(t, "req-1")
				addTargets(t, first.ID, "eu-west", "us-east")
				second := create(t, "req-2")
				addTargets(t, second.ID, "us-east", "ghost")
				targets, err := queries.ListTargetsOfPromotionRequest(ctx, first.ID)
				require.NoError(t, err)
				require.Equal(t, []string{"eu-west", "us-east"}, targetNames(targets))
				// A Target that does not exist is still part of the request's
				// fan-out, but not among the Targets it resolves to.
				targets, err = queries.ListTargetsOfPromotionRequest(ctx, second.ID)
				require.NoError(t, err)
				require.Equal(t, []string{"us-east"}, targetNames(targets))
				fanOut, err := queries.ListPromotionRequestTargets(ctx, []uuid.UUID{second.ID})
				require.NoError(t, err)
				require.Len(t, fanOut, 2)
				requests, err := queries.ListPromotionRequestsByTarget(ctx, ListPromotionRequestsByTargetParams{
					ProjectName: "demo",
					TargetName:  "us-east",
				})
				require.NoError(t, err)
				require.Equal(t, []string{"req-2", "req-1"}, requestNames(requests))
				requests, err = queries.ListPromotionRequestsByTarget(ctx, ListPromotionRequestsByTargetParams{
					ProjectName: "demo",
					TargetName:  "ap-south",
				})
				require.NoError(t, err)
				require.Empty(t, requests)
			},
		},
		{
			name: "deleting a Target leaves the requests that fanned out to it",
			run: func(t *testing.T) {
				createTarget(t, "us-east")
				row := create(t, "req-1")
				addTargets(t, row.ID, "us-east")
				deleted, err := queries.DeleteTarget(ctx, DeleteTargetParams{ProjectName: "demo", Name: "us-east"})
				require.NoError(t, err)
				require.EqualValues(t, 1, deleted)
				fanOut, err := queries.ListPromotionRequestTargets(ctx, []uuid.UUID{row.ID})
				require.NoError(t, err)
				require.Len(t, fanOut, 1)
				targets, err := queries.ListTargetsOfPromotionRequest(ctx, row.ID)
				require.NoError(t, err)
				require.Empty(t, targets)
			},
		},
		{
			name: "deleting a request takes its fan-out with it",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				addTargets(t, row.ID, "us-east")
				deleted, err := queries.DeletePromotionRequest(ctx, DeletePromotionRequestParams{
					ProjectName: "demo",
					Name:        "req-1",
				})
				require.NoError(t, err)
				require.EqualValues(t, 1, deleted)
				fanOut, err := queries.ListPromotionRequestTargets(ctx, []uuid.UUID{row.ID})
				require.NoError(t, err)
				require.Empty(t, fanOut)
				deleted, err = queries.DeletePromotionRequest(ctx, DeletePromotionRequestParams{
					ProjectName: "demo",
					Name:        "req-1",
				})
				require.NoError(t, err)
				require.Zero(t, deleted)
			},
		},
		{
			name: "a locked row blocks a second writer until the transaction ends",
			run: func(t *testing.T) {
				row := create(t, "req-1")
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = queries.WithTx(tx).GetPromotionRequestByIDForUpdate(ctx, row.ID)
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				_, err = queries.UpdatePromotionRequestStatus(deadlineCtx, UpdatePromotionRequestStatusParams{
					ID:    row.ID,
					Phase: "Running",
				})
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				_, err = queries.UpdatePromotionRequestStatus(ctx, UpdatePromotionRequestStatusParams{
					ID:    row.ID,
					Phase: "Running",
				})
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE promotion_requests, stages, targets CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func requestNames(rows []PromotionRequestRow) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.Name
	}
	return names
}

func targetNames(rows []TargetRow) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.Name
	}
	return names
}
