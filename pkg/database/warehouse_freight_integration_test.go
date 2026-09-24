//go:build integration

package database

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWarehouseFreightIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	store := NewStore(pool)
	queries := New(pool)
	ctx := context.Background()
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	project := UpsertProjectParams{ID: "project", Name: "demo", CreatedAt: created}
	warehouse := UpsertWarehouseParams{ID: "warehouse", ProjectID: project.ID, Name: "images", CreatedAt: created}
	testCases := []struct {
		name string
		run  func(*testing.T, FreightUpsert)
	}{
		{
			name: "out of order parents retry and cross project warehouses are rejected",
			run: func(t *testing.T, freight FreightUpsert) {
				requirePGError(t, store.UpsertWarehouse(ctx, warehouse), "23503")
				requirePGError(t, store.UpsertFreight(ctx, freight), "23503")
				require.NoError(t, store.UpsertProject(ctx, project))
				requirePGError(t, store.UpsertFreight(ctx, freight), "23503")
				require.NoError(t, store.UpsertWarehouse(ctx, warehouse))
				require.NoError(t, store.UpsertFreight(ctx, freight))
				require.NoError(t, store.UpsertProject(ctx, UpsertProjectParams{
					ID: "other", Name: "other", CreatedAt: created,
				}))
				freight.ID, freight.ProjectID = "other", "other"
				requirePGError(t, store.UpsertFreight(ctx, freight), "23503")
			},
		},
		{
			name: "repeated writes preserve artifact identities and exact metadata",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				_, err := pool.Exec(ctx, "UPDATE freight SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "UPDATE warehouses SET synced_at = '2000-01-01'")
				require.NoError(t, err)
				require.NoError(t, store.UpsertWarehouse(ctx, warehouse))
				require.NoError(t, store.UpsertFreight(ctx, freight))
				rows, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, freight.FreightContents, rows[0].FreightContents)
				require.True(t, rows[0].CreatedAt.Equal(created))
				require.True(t, rows[0].DiscoveredAt.Equal(created.Add(-time.Hour)))
				require.True(t, rows[0].SyncedAt.After(created))
				warehouses, err := store.ListWarehouses(ctx)
				require.NoError(t, err)
				require.Len(t, warehouses, 1)
				require.True(t, warehouses[0].CreatedAt.Equal(created))
				require.True(t, warehouses[0].SyncedAt.After(created))
				for _, table := range artifactTables() {
					_, err = pool.Exec(ctx, "INSERT INTO "+table+" SELECT * FROM "+table)
					requirePGError(t, err, "23505")
				}
			},
		},
		{
			name: "artifact ordinals preserve order and permit repeated artifact identities",
			run: func(t *testing.T, freight FreightUpsert) {
				freight.Commits = append(freight.Commits, freight.Commits[0])
				freight.Commits[1].Ordinal = 1
				freight.Images = append(freight.Images, freight.Images[0])
				freight.Images[1].Ordinal = 1
				freight.Charts = append(freight.Charts, freight.Charts[0])
				freight.Charts[1].Ordinal = 1
				freight.Artifacts = append(freight.Artifacts, freight.Artifacts[0])
				freight.Artifacts[1].Ordinal = 1
				freight.Commits[1].Message = "second commit"
				require.NoError(t, store.UpsertFreight(ctx, freight))
				slices.Reverse(freight.Commits)
				slices.Reverse(freight.Images)
				slices.Reverse(freight.Charts)
				slices.Reverse(freight.Artifacts)
				freight.Commits[0].Ordinal, freight.Commits[1].Ordinal = 0, 1
				freight.Images[0].Ordinal, freight.Images[1].Ordinal = 0, 1
				freight.Charts[0].Ordinal, freight.Charts[1].Ordinal = 0, 1
				freight.Artifacts[0].Ordinal, freight.Artifacts[1].Ordinal = 0, 1
				require.NoError(t, store.UpsertFreight(ctx, freight))
				after, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Equal(t, freight.FreightContents, after[0].FreightContents)
				require.Equal(t, "second commit", after[0].Commits[0].Message)
				for _, table := range artifactTables() {
					var count int
					require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count))
					require.Equal(t, 2, count)
				}
			},
		},
		{
			name: "artifact updates and removal are atomic",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				freight.Alias = "changed"
				freight.Commits[0].CommitID = "changed"
				freight.Images = nil
				freight.Charts = nil
				freight.Artifacts[0].Metadata = []byte("[]")
				requirePGError(t, store.UpsertFreight(ctx, freight), "23514")
				rows, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Equal(t, "friendly", rows[0].Alias)
				require.Equal(t, "abc", rows[0].Commits[0].CommitID)
				require.Len(t, rows[0].Images, 1)
				freight.Artifacts = nil
				require.NoError(t, store.UpsertFreight(ctx, freight))
				rows, err = store.ListFreight(ctx)
				require.NoError(t, err)
				require.Equal(t, freight.FreightContents, rows[0].FreightContents)
				freight.Commits = nil
				require.NoError(t, store.UpsertFreight(ctx, freight))
				rows, err = store.ListFreight(ctx)
				require.NoError(t, err)
				require.Equal(t, FreightContents{}, rows[0].FreightContents)
			},
		},
		{
			name: "warehouse recreation retains history and original freight association",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				replacement := warehouse
				replacement.ID = "new-warehouse"
				require.NoError(t, store.UpsertWarehouse(ctx, replacement))
				var deleted bool
				require.NoError(t, pool.QueryRow(ctx,
					"SELECT deleted_at IS NOT NULL FROM warehouses WHERE id = $1", warehouse.ID,
				).Scan(&deleted))
				require.True(t, deleted)
				freight.WarehouseID = replacement.ID
				require.NoError(t, store.UpsertFreight(ctx, freight))
				id, err := store.GetFreightWarehouseID(ctx, freight.ID)
				require.NoError(t, err)
				require.Equal(t, warehouse.ID, id)
				warehouses, err := store.ListWarehouses(ctx)
				require.NoError(t, err)
				require.Len(t, warehouses, 1)
				require.Equal(t, replacement.ID, warehouses[0].ID)
			},
		},
		{
			name: "freight recreation retains old artifacts and active name uniqueness",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				freight.ID = "replacement"
				requirePGError(t, queries.UpsertFreight(ctx, freight.UpsertFreightParams), "23505")
				require.NoError(t, store.UpsertFreight(ctx, freight))
				rows, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, freight.ID, rows[0].ID)
				old, err := queries.GetFreightByID(ctx, "freight")
				require.NoError(t, err)
				require.True(t, old.DeletedAt.Valid)
				for _, table := range artifactTables() {
					var count int
					require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count))
					require.Equal(t, 2, count)
				}
			},
		},
		{
			name: "failed replacement preserves the active identity",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				freight.ID = "replacement"
				freight.Artifacts[0].Metadata = []byte("false")
				requirePGError(t, store.UpsertFreight(ctx, freight), "23514")
				rows, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, "freight", rows[0].ID)
				_, err = pool.Exec(ctx,
					"ALTER TABLE warehouses ADD CONSTRAINT reject_replacement CHECK (id <> 'rejected')",
				)
				require.NoError(t, err)
				replacement := warehouse
				replacement.ID = "rejected"
				requirePGError(t, store.UpsertWarehouse(ctx, replacement), "23514")
				warehouses, err := store.ListWarehouses(ctx)
				require.NoError(t, err)
				require.Len(t, warehouses, 1)
				require.Equal(t, warehouse.ID, warehouses[0].ID)
			},
		},
		{
			name: "soft deletion is idempotent and same UID can be restored",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				require.NoError(t, store.DeleteFreightByIDs(ctx, []string{freight.ID}))
				require.NoError(t, store.DeleteWarehousesByID(ctx, []string{warehouse.ID}))
				before, err := queries.GetFreightByID(ctx, freight.ID)
				require.NoError(t, err)
				require.True(t, before.DeletedAt.Valid)
				require.NoError(t, store.DeleteFreightByIDs(ctx, []string{freight.ID}))
				after, err := queries.GetFreightByID(ctx, freight.ID)
				require.NoError(t, err)
				require.Equal(t, before.DeletedAt, after.DeletedAt)
				rows, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Empty(t, rows)
				warehouses, err := store.ListWarehouses(ctx)
				require.NoError(t, err)
				require.Empty(t, warehouses)
				require.NoError(t, store.UpsertWarehouse(ctx, warehouse))
				require.NoError(t, store.UpsertFreight(ctx, freight))
				rows, err = store.ListFreight(ctx)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.False(t, rows[0].DeletedAt.Valid)
				require.Equal(t, freight.FreightContents, rows[0].FreightContents)
			},
		},
		{
			name: "names and deletion are project scoped and project deletion cascades",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				require.NoError(t, store.UpsertProject(ctx, UpsertProjectParams{
					ID: "other", Name: "other", CreatedAt: created,
				}))
				otherWarehouse := warehouse
				otherWarehouse.ID, otherWarehouse.ProjectID = "other-warehouse", "other"
				require.NoError(t, store.UpsertWarehouse(ctx, otherWarehouse))
				freight.ID, freight.ProjectID, freight.WarehouseID = "other-freight", "other", otherWarehouse.ID
				require.NoError(t, store.UpsertFreight(ctx, freight))
				require.NoError(t, store.DeleteFreightByName(ctx, DeleteFreightByNameParams{
					ProjectName: project.Name, Name: freight.Name,
				}))
				require.NoError(t, store.DeleteWarehouseByName(ctx, DeleteWarehouseByNameParams{
					ProjectName: project.Name, Name: warehouse.Name,
				}))
				// Candidates captured before the second insert cannot delete it.
				require.NoError(t, store.DeleteFreightByIDs(ctx, []string{"freight"}))
				require.NoError(t, store.DeleteWarehousesByID(ctx, []string{warehouse.ID}))
				rows, err := store.ListFreight(ctx)
				require.NoError(t, err)
				require.Len(t, rows, 1)
				require.Equal(t, freight.ID, rows[0].ID)
				require.NoError(t, store.DeleteProjectsByID(ctx, []string{project.ID, "other"}))
				for _, table := range append(artifactTables(), "warehouses", "freight") {
					var count int
					require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count))
					require.Zero(t, count)
				}
			},
		},
		{
			name: "blocked freight transaction respects deadline and recovers",
			run: func(t *testing.T, freight FreightUpsert) {
				require.NoError(t, store.UpsertFreight(ctx, freight))
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = tx.Exec(ctx, "SELECT id FROM freight FOR UPDATE")
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				require.ErrorIs(t, store.UpsertFreight(deadlineCtx, freight), context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				require.NoError(t, store.UpsertFreight(ctx, freight))
			},
		},
	}
	for i, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE projects CASCADE")
			require.NoError(t, err)
			if i != 0 { // The first case explicitly exercises missing parents.
				require.NoError(t, store.UpsertProject(ctx, project))
				require.NoError(t, store.UpsertWarehouse(ctx, warehouse))
			}
			testCase.run(t, integrationFreight(created))
		})
	}
}

func TestCreationTimestampDefaultsIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	ctx := context.Background()
	for _, query := range []string{
		"INSERT INTO projects (id, name, created_at) VALUES ('p', 'demo', CURRENT_TIMESTAMP)",
		"INSERT INTO warehouses (id, project_id, name) VALUES ('w', 'p', 'images')",
		`INSERT INTO freight (id, project_id, warehouse_id, name, alias, discovered_at)
		 VALUES ('f', 'p', 'w', 'bundle', '', CURRENT_TIMESTAMP)`,
	} {
		_, err := pool.Exec(ctx, query)
		require.NoError(t, err)
	}
	for _, table := range []string{"warehouses", "freight"} {
		var created time.Time
		require.NoError(t, pool.QueryRow(ctx, "SELECT created_at FROM "+table).Scan(&created))
		require.WithinDuration(t, time.Now(), created, time.Minute)
		_, err := pool.Exec(ctx, "UPDATE "+table+" SET created_at = NULL")
		requirePGError(t, err, "23502")
	}
}

func TestArtifactSchemaIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := isolatedDatabase(t)
	ctx := context.Background()
	for _, table := range artifactTables() {
		var columns string
		err := pool.QueryRow(ctx, `
			SELECT string_agg(a.attname, ',' ORDER BY k.ordinality)
			FROM pg_constraint c
			CROSS JOIN LATERAL unnest(c.conkey) WITH ORDINALITY k(attnum, ordinality)
			JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
			WHERE c.conrelid = $1::regclass AND c.contype = 'p'`, table).Scan(&columns)
		require.NoError(t, err)
		require.Equal(t, "freight_id,ordinal", columns)
	}
	for table, column := range map[string]string{
		"freight_images": "annotations", "freight_artifacts": "metadata",
	} {
		var columnType string
		err := pool.QueryRow(ctx, `
			SELECT format_type(atttypid, atttypmod)
			FROM pg_attribute WHERE attrelid = $1::regclass AND attname = $2`, table, column).Scan(&columnType)
		require.NoError(t, err)
		require.Equal(t, "jsonb", columnType)
	}
}

func artifactTables() []string {
	return []string{"freight_commits", "freight_images", "freight_charts", "freight_artifacts"}
}

func integrationFreight(created time.Time) FreightUpsert {
	return FreightUpsert{
		UpsertFreightParams: UpsertFreightParams{
			ID: "freight", ProjectID: "project", WarehouseID: "warehouse", Name: "bundle", Alias: "friendly",
			CreatedAt: created, DiscoveredAt: created.Add(-time.Hour),
		},
		FreightContents: FreightContents{
			Commits: []UpsertFreightCommitParams{{FreightID: "freight", RepoURL: "https://example.com/git", CommitID: "abc"}},
			Images: []UpsertFreightImageParams{{
				FreightID: "freight", RepoURL: "example.com/image", Tag: "v1", Digest: "sha256:abc",
				Annotations: []byte(`{"build": "123"}`),
			}},
			Charts: []UpsertFreightChartParams{{
				FreightID: "freight", RepoURL: "https://example.com/charts", Name: "app", Version: "1.0",
			}},
			Artifacts: []UpsertFreightArtifactParams{{
				FreightID: "freight", ArtifactType: "custom", SubscriptionName: "custom", Version: "v1",
				Metadata: []byte(`{"number": 9007199254740993}`),
			}},
		},
	}
}
