//go:build integration

package database

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// seedMirror writes the mirrored rows a PromotionRequest depends on: a
// Project, a Stage, a Warehouse and one piece of Freight.
func seedMirror(ctx context.Context, t *testing.T, store Store) {
	t.Helper()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, store.UpsertProject(ctx, UpsertProjectParams{ID: "project", Name: "demo", CreatedAt: created}))
	require.NoError(t, store.UpsertStage(ctx, UpsertStageParams{
		ID: "stage", ProjectID: "project", Name: "dev", CreatedAt: created,
	}))
	require.NoError(t, store.UpsertWarehouse(ctx, UpsertWarehouseParams{
		ID: "warehouse", ProjectID: "project", Name: "images", CreatedAt: created,
	}))
	require.NoError(t, store.UpsertFreight(ctx, FreightUpsert{UpsertFreightParams: UpsertFreightParams{
		ID: "freight", ProjectID: "project", WarehouseID: "warehouse", Name: "abcdefg1234",
		Alias: "witty-wombat", DiscoveredAt: created, CreatedAt: created,
	}}))
}

// seedTarget inserts a Target the way an operator would, with plain SQL.
func seedTarget(ctx context.Context, t *testing.T, pool *pgxpool.Pool, name, labels, params string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO targets (project_id, name, labels, params)
		SELECT id, $1, $2::jsonb, $3::jsonb FROM projects WHERE name = 'demo'`, name, labels, params)
	require.NoError(t, err)
}

func TestTargetsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	store := NewStore(pool)
	ctx := context.Background()
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "targets need a project",
			run: func(t *testing.T) {
				_, err := pool.Exec(ctx, `INSERT INTO targets (project_id, name) VALUES ('missing', 'x')`)
				requirePGError(t, err, "23503")
			},
		},
		{
			name: "labels and params must be objects",
			run: func(t *testing.T) {
				seedMirror(ctx, t, store)
				_, err := pool.Exec(ctx, `INSERT INTO targets (project_id, name, labels)
					SELECT id, 'x', '[]'::jsonb FROM projects`)
				requirePGError(t, err, "23514")
				_, err = pool.Exec(ctx, `INSERT INTO targets (project_id, name, params)
					SELECT id, 'x', '"str"'::jsonb FROM projects`)
				requirePGError(t, err, "23514")
			},
		},
		{
			name: "names are unique within a project",
			run: func(t *testing.T) {
				seedMirror(ctx, t, store)
				seedTarget(ctx, t, pool, "us-east", `{}`, `{}`)
				_, err := pool.Exec(ctx, `INSERT INTO targets (project_id, name)
					SELECT id, 'us-east' FROM projects`)
				requirePGError(t, err, "23505")
			},
		},
		{
			name: "list is ordered by name and get returns exact values",
			run: func(t *testing.T) {
				seedMirror(ctx, t, store)
				seedTarget(ctx, t, pool, "eu-west", `{"region":"eu"}`, `{"replicas":2}`)
				seedTarget(ctx, t, pool, "us-east", `{"region":"us"}`, `{}`)
				targets, err := store.ListTargets(ctx, "demo")
				require.NoError(t, err)
				require.Len(t, targets, 2)
				require.Equal(t, "eu-west", targets[0].Name)
				require.Equal(t, "us-east", targets[1].Name)
				require.NotEqual(t, uuid.Nil, targets[0].ID)
				require.Equal(t, "project", targets[0].ProjectID)
				require.JSONEq(t, `{"region":"eu"}`, string(targets[0].Labels))
				require.JSONEq(t, `{"replicas":2}`, string(targets[0].Params))
				require.False(t, targets[0].CreatedAt.IsZero())
				require.False(t, targets[0].UpdatedAt.IsZero())

				target, err := store.GetTarget(ctx, GetTargetParams{ProjectName: "demo", Name: "us-east"})
				require.NoError(t, err)
				require.Equal(t, targets[1].ID, target.ID)

				_, err = store.GetTarget(ctx, GetTargetParams{ProjectName: "demo", Name: "nope"})
				require.ErrorIs(t, err, ErrNotFound)
				_, err = store.GetTarget(ctx, GetTargetParams{ProjectName: "other", Name: "us-east"})
				require.ErrorIs(t, err, ErrNotFound)

				other, err := store.ListTargets(ctx, "other")
				require.NoError(t, err)
				require.Empty(t, other)
			},
		},
		{
			name: "deleting the project removes its targets",
			run: func(t *testing.T) {
				seedMirror(ctx, t, store)
				seedTarget(ctx, t, pool, "us-east", `{}`, `{}`)
				require.NoError(t, store.DeleteProjectByName(ctx, "demo"))
				var count int
				require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM targets").Scan(&count))
				require.Zero(t, count)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE projects CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func TestPromotionRequestsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	store := NewStore(pool)
	ctx := context.Background()
	newCreate := func(name string, targets ...string) PromotionRequestCreate {
		return PromotionRequestCreate{
			CreatePromotionRequestParams: CreatePromotionRequestParams{
				ProjectName: "demo", Stage: "dev", Freight: "abcdefg1234", Name: name, CreatedBy: "user:someone",
			},
			Targets: targets,
		}
	}
	seedAll := func(t *testing.T) {
		seedMirror(ctx, t, store)
		seedTarget(ctx, t, pool, "eu-west", `{"region":"eu"}`, `{}`)
		seedTarget(ctx, t, pool, "us-east", `{"region":"us"}`, `{}`)
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "creation waits for the mirror",
			run: func(t *testing.T) {
				_, err := store.CreatePromotionRequest(ctx, newCreate("dev.1"))
				require.ErrorIs(t, err, ErrNotMirrored)
				seedMirror(ctx, t, store)
				_, err = store.CreatePromotionRequest(ctx, newCreate("dev.1", "us-east"))
				require.ErrorIs(t, err, ErrNotMirrored)
				require.ErrorContains(t, err, `Target "us-east"`)
				// The failed attempt left nothing behind.
				var count int
				require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM promotion_requests").Scan(&count))
				require.Zero(t, count)
				seedTarget(ctx, t, pool, "us-east", `{}`, `{}`)
				_, err = store.CreatePromotionRequest(ctx, newCreate("dev.1", "us-east"))
				require.NoError(t, err)
			},
		},
		{
			name: "creation resolves references and records targets in order",
			run: func(t *testing.T) {
				seedAll(t)
				snapshot, err := store.CreatePromotionRequest(ctx, newCreate("dev.1", "us-east", "eu-west"))
				require.NoError(t, err)
				require.NotEqual(t, uuid.Nil, snapshot.ID)
				require.Positive(t, snapshot.Seq)
				require.Equal(t, "project", snapshot.ProjectID)
				require.Equal(t, "stage", snapshot.StageID)
				require.Equal(t, "freight", snapshot.FreightID)
				require.Equal(t, "demo", snapshot.ProjectName)
				require.Equal(t, "dev", snapshot.Stage)
				require.Equal(t, "abcdefg1234", snapshot.Freight)
				require.Equal(t, "dev.1", snapshot.Name)
				require.Equal(t, "user:someone", snapshot.CreatedBy)
				require.Equal(t, "Pending", snapshot.Phase)
				require.Empty(t, snapshot.Message)
				require.False(t, snapshot.CreatedAt.IsZero())
				require.False(t, snapshot.StartedAt.Valid)
				require.False(t, snapshot.FinishedAt.Valid)
				require.Len(t, snapshot.Targets, 2)
				require.Equal(t, "us-east", snapshot.Targets[0].Name)
				require.Equal(t, int64(0), snapshot.Targets[0].Ordinal)
				require.Equal(t, "eu-west", snapshot.Targets[1].Name)
				require.Equal(t, int64(1), snapshot.Targets[1].Ordinal)

				// Reading it back yields the same snapshot, targets in order.
				read, err := store.GetPromotionRequest(ctx, GetPromotionRequestParams{ProjectName: "demo", Name: "dev.1"})
				require.NoError(t, err)
				require.Equal(t, snapshot, read)

				// Names are unique per project; a soft-deleted Freight is not a valid reference.
				_, err = store.CreatePromotionRequest(ctx, newCreate("dev.1"))
				requirePGError(t, err, "23505")
				require.NoError(t, store.DeleteFreightByName(ctx, DeleteFreightByNameParams{
					ProjectName: "demo", Name: "abcdefg1234",
				}))
				_, err = store.CreatePromotionRequest(ctx, newCreate("dev.2"))
				require.ErrorIs(t, err, ErrNotMirrored)
			},
		},
		{
			name: "lists are ordered by seq and scoped by project and stage",
			run: func(t *testing.T) {
				seedAll(t)
				require.NoError(t, store.UpsertStage(ctx, UpsertStageParams{
					ID: "stage-2", ProjectID: "project", Name: "prod", CreatedAt: time.Now(),
				}))
				// Names deliberately sort against creation order.
				for _, name := range []string{"dev.c", "dev.b", "dev.a"} {
					_, err := store.CreatePromotionRequest(ctx, newCreate(name, "us-east"))
					require.NoError(t, err)
				}
				prod := newCreate("prod.a")
				prod.Stage = "prod"
				_, err := store.CreatePromotionRequest(ctx, prod)
				require.NoError(t, err)

				all, err := store.ListPromotionRequests(ctx, "demo")
				require.NoError(t, err)
				require.Equal(t, []string{"dev.c", "dev.b", "dev.a", "prod.a"}, names(all))
				require.True(t, all[0].Seq < all[1].Seq && all[1].Seq < all[2].Seq && all[2].Seq < all[3].Seq)
				require.Len(t, all[0].Targets, 1)
				require.Empty(t, all[3].Targets)

				dev, err := store.ListPromotionRequestsByStage(ctx, ListPromotionRequestsByStageParams{
					ProjectName: "demo", Stage: "dev",
				})
				require.NoError(t, err)
				require.Equal(t, []string{"dev.c", "dev.b", "dev.a"}, names(dev))

				other, err := store.ListPromotionRequests(ctx, "other")
				require.NoError(t, err)
				require.Empty(t, other)

				_, err = store.GetPromotionRequest(ctx, GetPromotionRequestParams{ProjectName: "other", Name: "dev.a"})
				require.ErrorIs(t, err, ErrNotFound)

				exists, err := store.PromotionRequestExists(ctx, PromotionRequestExistsParams{
					ProjectName: "demo", Stage: "dev", Freight: "abcdefg1234",
				})
				require.NoError(t, err)
				require.True(t, exists)
				exists, err = store.PromotionRequestExists(ctx, PromotionRequestExistsParams{
					ProjectName: "demo", Stage: "dev", Freight: "other",
				})
				require.NoError(t, err)
				require.False(t, exists)
			},
		},
		{
			name: "status updates return both versions and leave the open set",
			run: func(t *testing.T) {
				seedAll(t)
				var ids []uuid.UUID
				for _, name := range []string{"dev.1", "dev.2", "dev.3"} {
					snapshot, err := store.CreatePromotionRequest(ctx, newCreate(name, "us-east", "eu-west"))
					require.NoError(t, err)
					ids = append(ids, snapshot.ID)
				}
				first, err := store.GetPromotionRequestByID(ctx, ids[0])
				require.NoError(t, err)
				require.Equal(t, "dev.1", first.Name)
				require.Equal(t, "demo", first.ProjectName)
				require.Len(t, first.Targets, 2)

				started := metav1.NewTime(time.Date(2021, 6, 7, 8, 9, 11, 0, time.UTC))
				finished := metav1.NewTime(started.Add(time.Minute))
				before, after, err := store.UpdatePromotionRequestStatus(ctx, first.ID, kargoapi.PromotionRequestStatus{
					Phase:      kargoapi.PromotionRequestPhaseSucceeded,
					Message:    "all done",
					StartedAt:  &started,
					FinishedAt: &finished,
					Targets: []kargoapi.PromotionRequestTargetStatus{
						{Name: "us-east", Promotion: "dev.us-east.1", Phase: kargoapi.PromotionPhaseSucceeded},
						{Name: "not-a-target", Promotion: "x", Phase: kargoapi.PromotionPhaseFailed},
					},
				})
				require.NoError(t, err)
				// Before is exactly what was there.
				require.Equal(t, first, before)
				// After is what the write left, and what a fresh read finds.
				require.Equal(t, "Succeeded", after.Phase)
				require.Equal(t, "all done", after.Message)
				require.True(t, after.StartedAt.Valid && after.StartedAt.Time.Equal(started.Time))
				require.True(t, after.FinishedAt.Valid && after.FinishedAt.Time.Equal(finished.Time))
				require.True(t, after.UpdatedAt.After(before.UpdatedAt))
				require.Equal(t, "dev.us-east.1", after.Targets[0].Promotion)
				require.Equal(t, "Succeeded", after.Targets[0].Phase)
				require.Empty(t, after.Targets[1].Promotion)
				require.Empty(t, after.Targets[1].Phase)
				reread, err := store.GetPromotionRequestByID(ctx, first.ID)
				require.NoError(t, err)
				require.Equal(t, after, reread)
				request := PromotionRequestFromSnapshot(after)
				require.Equal(t, &kargoapi.PromotionRequestSummary{Succeeded: 1}, request.Status.Summary)

				// The terminal request has left the open set; the rest page by id.
				open := []uuid.UUID{ids[1], ids[2]}
				slices.SortFunc(open, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
				page, err := store.ListOpenPromotionRequestIDs(ctx, uuid.Nil, 1)
				require.NoError(t, err)
				require.Equal(t, open[:1], page)
				page, err = store.ListOpenPromotionRequestIDs(ctx, page[0], 10)
				require.NoError(t, err)
				require.Equal(t, open[1:], page)
				_, err = store.ListOpenPromotionRequestIDs(ctx, uuid.Nil, 0)
				require.ErrorContains(t, err, "invalid limit 0")

				// Unknown ids, and phases the schema does not know, are rejected.
				_, err = store.GetPromotionRequestByID(ctx, uuid.New())
				require.ErrorIs(t, err, ErrNotFound)
				_, _, err = store.UpdatePromotionRequestStatus(ctx, uuid.New(), kargoapi.PromotionRequestStatus{
					Phase: kargoapi.PromotionRequestPhaseErrored,
				})
				require.ErrorIs(t, err, ErrNotFound)
				_, _, err = store.UpdatePromotionRequestStatus(ctx, first.ID, kargoapi.PromotionRequestStatus{
					Phase: "Bogus",
				})
				requirePGError(t, err, "23514")
			},
		},
		{
			name: "deleting a stage, freight, target or project cascades",
			run: func(t *testing.T) {
				seedAll(t)
				_, err := store.CreatePromotionRequest(ctx, newCreate("dev.1", "us-east", "eu-west"))
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "DELETE FROM targets WHERE name = 'eu-west'")
				require.NoError(t, err)
				snapshot, err := store.GetPromotionRequest(ctx, GetPromotionRequestParams{ProjectName: "demo", Name: "dev.1"})
				require.NoError(t, err)
				require.Equal(t, []string{"us-east"}, targetNames(snapshot))

				require.NoError(t, store.DeleteStageByName(ctx, DeleteStageByNameParams{ProjectName: "demo", Name: "dev"}))
				_, err = store.GetPromotionRequest(ctx, GetPromotionRequestParams{ProjectName: "demo", Name: "dev.1"})
				require.ErrorIs(t, err, ErrNotFound)
				var count int
				require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM promotion_request_targets").Scan(&count))
				require.Zero(t, count)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE projects CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func names(snapshots []PromotionRequestSnapshot) []string {
	out := make([]string, len(snapshots))
	for i, snapshot := range snapshots {
		out[i] = snapshot.Name
	}
	return out
}

func targetNames(snapshot PromotionRequestSnapshot) []string {
	out := make([]string, len(snapshot.Targets))
	for i, target := range snapshot.Targets {
		out[i] = target.Name
	}
	return out
}
