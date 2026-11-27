//go:build integration && db

package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database/databasetest"
)

// TestPromotionRolloutStageKeysMigrationIntegration covers the data the
// migration that keys rollouts by their Stage's row finds already there.
func TestPromotionRolloutStageKeysMigrationIntegration(t *testing.T) {
	t.Parallel()
	pool, migrations := databasetest.IsolatedDatabase(t)
	ctx := context.Background()
	// The last version before rollouts were keyed by their Stage's row.
	const before = 20261104000000
	_, err := migrations.DownTo(ctx, before)
	require.NoError(t, err)

	// prod was recreated while rollouts still followed the Stage's name, so
	// its new row's counter is behind the rollouts it inherited. staging's
	// counter is ahead of its rollouts.
	_, err = pool.Exec(ctx, `
		INSERT INTO stages (id, project_name, name, last_promotion_rollout_number, created_at)
		VALUES
			('prod-uid', 'demo', 'prod', 0, now()),
			('staging-uid', 'demo', 'staging', 5, now())`,
	)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO promotion_rollouts (project_name, name, stage_id, stage_name, number, freight_name)
		VALUES
			('demo', 'prod.a', 'prod-uid', 'prod', 1, 'f'),
			('demo', 'prod.b', 'prod-uid', 'prod', 2, 'f'),
			('demo', 'staging.a', 'staging-uid', 'staging', 1, 'f')`,
	)
	require.NoError(t, err)

	_, err = migrations.Up(ctx)
	require.NoError(t, err)

	// Each Stage's next rollout follows both its counter and the rollouts it
	// already has.
	queries := New(pool)
	for stage, want := range map[string]int64{"prod": 3, "staging": 6} {
		rollout, createErr := queries.CreatePromotionRollout(ctx, CreatePromotionRolloutParams{
			ProjectName: "demo",
			StageName:   stage,
			Name:        stage + ".next",
			FreightName: "f",
		})
		require.NoError(t, createErr)
		require.Equal(t, want, rollout.Number, stage)
	}
}

// TestPromotionRolloutTargetProjectMigrationIntegration covers the data the
// migration that holds a rollout's Targets to its Project finds already
// there, and what its down step leaves.
func TestPromotionRolloutTargetProjectMigrationIntegration(t *testing.T) {
	t.Parallel()
	pool, migrations := databasetest.IsolatedDatabase(t)
	ctx := context.Background()
	// The last version before a rollout's Targets were held to its Project.
	const before = 20261105000000
	_, err := migrations.DownTo(ctx, before)
	require.NoError(t, err)

	// A rollout of demo recorded with one of demo's Targets and, as nothing
	// prevented, one of another Project's.
	_, err = pool.Exec(ctx, `
		INSERT INTO stages (id, project_name, name, last_promotion_rollout_number, created_at)
		VALUES ('prod-uid', 'demo', 'prod', 1, now())`,
	)
	require.NoError(t, err)
	var rolloutID, ownID, foreignID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO promotion_rollouts (project_name, name, stage_id, stage_name, number, freight_name)
		VALUES ('demo', 'prod.a', 'prod-uid', 'prod', 1, 'f')
		RETURNING id`,
	).Scan(&rolloutID))
	const insertTarget = `
		INSERT INTO targets (project_name, name, target_set_name)
		VALUES ($1, 'us-east', 'fleet')
		RETURNING id`
	require.NoError(t, pool.QueryRow(ctx, insertTarget, "demo").Scan(&ownID))
	require.NoError(t, pool.QueryRow(ctx, insertTarget, "other").Scan(&foreignID))
	_, err = pool.Exec(ctx, `
		INSERT INTO promotion_rollout_targets (promotion_rollout_id, target_id)
		VALUES ($1, $2), ($1, $3)`,
		rolloutID, ownID, foreignID,
	)
	require.NoError(t, err)

	_, err = migrations.Up(ctx)
	require.NoError(t, err)

	// The rollout keeps its own Project's Target, now recorded under the
	// Project, and loses the other's.
	queries := New(pool)
	recorded, err := queries.ListPromotionRolloutTargets(ctx, []uuid.UUID{rolloutID})
	require.NoError(t, err)
	require.Len(t, recorded, 1)
	require.Equal(t, ownID, recorded[0].TargetID)
	var project string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT project_name FROM promotion_rollout_targets WHERE promotion_rollout_id = $1",
		rolloutID,
	).Scan(&project))
	require.Equal(t, "demo", project)
	// The other Project's Target itself is untouched.
	var remaining int
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT count(*) FROM targets WHERE id = $1", foreignID,
	).Scan(&remaining))
	require.Equal(t, 1, remaining)

	// The down step restores the references by id alone, which still remove
	// a snapshot row with its Target, and the migration applies again over
	// what it left.
	_, err = migrations.DownTo(ctx, before)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO promotion_rollout_targets (promotion_rollout_id, target_id)
		VALUES ($1, $2)`,
		rolloutID, foreignID,
	)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM targets WHERE id = $1", foreignID)
	require.NoError(t, err)
	_, err = migrations.Up(ctx)
	require.NoError(t, err)
	recorded, err = queries.ListPromotionRolloutTargets(ctx, []uuid.UUID{rolloutID})
	require.NoError(t, err)
	require.Len(t, recorded, 1)
	require.Equal(t, ownID, recorded[0].TargetID)
}
