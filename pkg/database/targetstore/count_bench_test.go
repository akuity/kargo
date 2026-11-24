//go:build integration && db

package targetstore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
)

// BenchmarkCount counts a Project's Targets as its rollout history grows. The
// Project has 10 target-aware Stages of 10 Targets each; every Stage has one
// Running rollout above its finished ones, and another Project's Stages of
// the same names have as many rollouts again.
func BenchmarkCount(b *testing.B) {
	for _, rollouts := range []int{100, 10_000, 50_000} {
		b.Run(fmt.Sprintf("rollouts-%d", rollouts), func(b *testing.B) {
			pool, _ := databasetest.IsolatedDatabase(b)
			ctx := b.Context()
			_, err := pool.Exec(ctx, fmt.Sprintf(`
				INSERT INTO stages (
					id, project_name, name, target_selectors, last_promotion_rollout_number, created_at
				)
				SELECT project || '/s' || i, project, 's' || i, '[{}]', %[1]d + 1, now()
				FROM unnest(ARRAY['bench', 'other']) AS project, generate_series(0, 9) AS i;

				INSERT INTO targets (project_name, name, target_set_name)
				SELECT 'bench', 't' || i, 'fleet' FROM generate_series(0, 99) AS i;

				INSERT INTO promotion_rollouts (
					project_name, name, stage_id, stage_name, number, freight_name, status_phase
				)
				SELECT stages.project_name, stages.name || '.' || n, stages.id, stages.name, n,
					'freight',
					CASE
						WHEN n > %[1]d THEN 'Running'
						WHEN mod(n, 10) = 0 THEN 'Failed'
						ELSE 'Succeeded'
					END
				FROM stages, generate_series(1, %[1]d + 1) AS n;

				-- Stage si governs Targets t(10i) to t(10i+9).
				INSERT INTO promotion_rollout_targets (promotion_rollout_id, target_id)
				SELECT promotion_rollouts.id, targets.id
				FROM promotion_rollouts
				JOIN targets ON targets.project_name = promotion_rollouts.project_name
					AND substr(targets.name, 2)::int / 10 = substr(promotion_rollouts.stage_name, 2)::int;

				-- Each Running rollout has promoted to half its Targets.
				INSERT INTO promotions (
					id, project_name, name, stage_id, stage_name, freight_name,
					target_name, promotion_rollout_id, phase, created_at
				)
				SELECT promotion_rollouts.name || '.' || targets.name, 'bench',
					promotion_rollouts.name || '.' || targets.name, promotion_rollouts.stage_id,
					promotion_rollouts.stage_name, 'freight', targets.name, promotion_rollouts.id,
					'Succeeded', now()
				FROM promotion_rollouts
				JOIN promotion_rollout_targets
					ON promotion_rollout_targets.promotion_rollout_id = promotion_rollouts.id
				JOIN targets ON targets.id = promotion_rollout_targets.target_id
				WHERE promotion_rollouts.project_name = 'bench'
					AND promotion_rollouts.status_phase = 'Running'
					AND mod(substr(targets.name, 2)::int, 10) < 5;

				ANALYZE;`,
				rollouts/10,
			))
			require.NoError(b, err)
			store := New(database.New(pool))
			counts, err := store.Count(ctx, "bench")
			require.NoError(b, err)
			require.Equal(b, database.CountTargetsRow{Total: 100, Healthy: 50}, counts)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := store.Count(ctx, "bench"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
