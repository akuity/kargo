//go:build integration && db

package database

import (
	"context"
	"testing"

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
