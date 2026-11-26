//go:build integration && db

package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestProjectConfigQueries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	insert := func(
		t *testing.T,
		q *Queries,
		project string,
		uid string,
		spec string,
	) ProjectConfigRow {
		t.Helper()
		row, err := q.InsertProjectConfig(ctx, InsertProjectConfigParams{
			ProjectName: project,
			ProjectUID:  uid,
			Labels:      []byte(`{}`),
			Annotations: []byte(`{}`),
			Spec:        []byte(spec),
		})
		require.NoError(t, err)
		return row
	}
	specParams := func(spec string) UpdateProjectConfigSpecParams {
		return UpdateProjectConfigSpecParams{
			ProjectName: "p",
			Labels:      []byte(`{}`),
			Annotations: []byte(`{}`),
			Spec:        []byte(spec),
		}
	}
	receiverParams := func(
		p InsertProjectConfigWebhookReceiverParams,
	) InsertProjectConfigWebhookReceiverParams {
		p.Name = "r"
		p.URL = "https://example.com" + p.Path
		return p
	}
	testCases := []struct {
		name string
		run  func(*testing.T, *Queries)
	}{
		{
			name: "generation bumps only when spec changes",
			run: func(t *testing.T, q *Queries) {
				row := insert(t, q, "p", "uid", `{"a":1,"b":2}`)
				require.Equal(t, int64(1), row.Generation)
				same, err := q.UpdateProjectConfigSpec(
					ctx,
					specParams(`{"b":2,"a":1}`),
				)
				require.NoError(t, err)
				require.Equal(t, int64(1), same.Generation)
				require.True(t, same.UpdatedAt.After(row.UpdatedAt))
				changed, err := q.UpdateProjectConfigSpec(
					ctx,
					specParams(`{"a":2}`),
				)
				require.NoError(t, err)
				require.Equal(t, int64(2), changed.Generation)
				require.True(t, changed.UpdatedAt.After(same.UpdatedAt))
			},
		},
		{
			name: "update honors id and updated_at preconditions",
			run: func(t *testing.T, q *Queries) {
				row := insert(t, q, "p", "uid", `{}`)
				stale := row.UpdatedAt.Add(-time.Second)
				params := specParams(`{"a":1}`)
				params.UpdatedAt = &stale
				_, err := q.UpdateProjectConfigSpec(ctx, params)
				require.ErrorIs(t, err, pgx.ErrNoRows)
				params.ID = &row.ID
				params.UpdatedAt = &row.UpdatedAt
				_, err = q.UpdateProjectConfigSpec(ctx, params)
				require.NoError(t, err)
			},
		},
		{
			name: "duplicate path skipped",
			run: func(t *testing.T, q *Queries) {
				first := insert(t, q, "p1", "uid1", `{}`)
				second := insert(t, q, "p2", "uid2", `{}`)
				name, err := q.InsertProjectConfigWebhookReceiver(
					ctx,
					receiverParams(InsertProjectConfigWebhookReceiverParams{
						ProjectConfigID: first.ID,
						Path:            "/shared",
					}),
				)
				require.NoError(t, err)
				require.Equal(t, "r", name)
				_, err = q.InsertProjectConfigWebhookReceiver(
					ctx,
					receiverParams(InsertProjectConfigWebhookReceiverParams{
						ProjectConfigID: second.ID,
						Path:            "/shared",
					}),
				)
				require.ErrorIs(t, err, pgx.ErrNoRows)
			},
		},
		{
			name: "receivers cascade",
			run: func(t *testing.T, q *Queries) {
				row := insert(t, q, "p", "uid", `{}`)
				_, err := q.InsertProjectConfigWebhookReceiver(
					ctx,
					receiverParams(InsertProjectConfigWebhookReceiverParams{
						ProjectConfigID: row.ID,
						Path:            "/p",
					}),
				)
				require.NoError(t, err)
				receivers, err := q.ListProjectConfigWebhookReceivers(ctx, row.ID)
				require.NoError(t, err)
				require.Len(t, receivers, 1)
				deleted, err := q.DeleteProjectConfigByID(ctx, row.ID)
				require.NoError(t, err)
				require.Equal(t, row.ID, deleted.ID)
				receivers, err = q.ListProjectConfigWebhookReceivers(ctx, row.ID)
				require.NoError(t, err)
				require.Empty(t, receivers)
			},
		},
		{
			name: "leftover filter",
			run: func(t *testing.T, q *Queries) {
				require.NoError(t, q.UpsertProject(ctx, UpsertProjectParams{
					ID:        "uid-new",
					Name:      "p",
					CreatedAt: time.Now(),
				}))
				leftover := insert(t, q, "p", "uid-old", `{}`)
				_, err := q.InsertProjectConfigWebhookReceiver(
					ctx,
					receiverParams(InsertProjectConfigWebhookReceiverParams{
						ProjectConfigID: leftover.ID,
						Path:            "/leftover",
					}),
				)
				require.NoError(t, err)
				insert(t, q, "unmirrored", "any-uid", `{}`)
				insert(t, q, "q", "uid-q", `{}`)
				require.NoError(t, q.UpsertProject(ctx, UpsertProjectParams{
					ID:        "uid-q",
					Name:      "q",
					CreatedAt: time.Now(),
				}))

				rows, err := q.ListProjectConfigs(ctx)
				require.NoError(t, err)
				names := make([]string, 0, len(rows))
				for _, r := range rows {
					names = append(names, r.ProjectName)
				}
				require.Equal(t, []string{"q", "unmirrored"}, names)

				_, err = q.GetProjectConfigByWebhookReceiverPath(ctx, "/leftover")
				require.ErrorIs(t, err, pgx.ErrNoRows)

				// Get reads by name alone and leaves UID filtering to the store.
				got, err := q.GetProjectConfig(ctx, "p")
				require.NoError(t, err)
				require.Equal(t, "uid-old", got.ProjectUID)
			},
		},
		{
			name: "status rejected for stale generation or token",
			run: func(t *testing.T, q *Queries) {
				inserted := insert(t, q, "p", "uid", `{"a":1}`)
				updated, err := q.UpdateProjectConfigSpec(ctx, specParams(`{"a":2}`))
				require.NoError(t, err)
				status := func(
					generation int64,
					observedToken *string,
				) (ProjectConfigRow, error) {
					return q.UpdateProjectConfigStatus(
						ctx,
						UpdateProjectConfigStatusParams{
							StatusConditions:         []byte(`[]`),
							StatusObservedGeneration: generation,
							StatusLastHandledRefresh: observedToken,
							ProjectName:              "p",
							ID:                       inserted.ID,
							ObservedGeneration:       generation,
							ObservedRefreshToken:     observedToken,
						},
					)
				}
				// Never refreshed: the stored token is NULL and the observed
				// token is nil.
				fresh, err := status(2, nil)
				require.NoError(t, err)
				require.Nil(t, fresh.StatusLastHandledRefresh)
				require.True(t, fresh.UpdatedAt.After(updated.UpdatedAt))
				token := "new"
				refreshed, err := q.SetProjectConfigRefreshToken(
					ctx,
					SetProjectConfigRefreshTokenParams{
						ProjectName:  "p",
						RefreshToken: token,
					},
				)
				require.NoError(t, err)
				require.True(t, refreshed.UpdatedAt.After(fresh.UpdatedAt))
				_, err = status(1, &token)
				require.ErrorIs(t, err, pgx.ErrNoRows)
				stale := "old"
				_, err = status(2, &stale)
				require.ErrorIs(t, err, pgx.ErrNoRows)
				_, err = status(2, nil)
				require.ErrorIs(t, err, pgx.ErrNoRows)
				row, err := status(2, &token)
				require.NoError(t, err)
				require.Equal(t, int64(2), *row.StatusObservedGeneration)
				require.Equal(t, &token, row.StatusLastHandledRefresh)
				require.True(t, row.UpdatedAt.After(refreshed.UpdatedAt))
			},
		},
		{
			name: "database enforces unique project names and object spec",
			run: func(t *testing.T, q *Queries) {
				insert(t, q, "p", "uid", `{}`)
				_, err := q.InsertProjectConfig(ctx, InsertProjectConfigParams{
					ProjectName: "p",
					ProjectUID:  "other",
					Labels:      []byte(`{}`),
					Annotations: []byte(`{}`),
					Spec:        []byte(`{}`),
				})
				databasetest.RequirePGError(t, err, "23505")
				_, err = q.InsertProjectConfig(ctx, InsertProjectConfigParams{
					ProjectName: "other",
					ProjectUID:  "uid",
					Labels:      []byte(`{}`),
					Annotations: []byte(`{}`),
					Spec:        []byte(`[]`),
				})
				databasetest.RequirePGError(t, err, "23514")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			pool, _ := databasetest.IsolatedDatabase(t)
			testCase.run(t, New(pool))
		})
	}
}
