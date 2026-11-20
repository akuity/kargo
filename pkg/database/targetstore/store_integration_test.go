//go:build integration && db

package targetstore

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
)

func TestStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	store := New(queries)
	ctx := context.Background()
	newTarget := func(name string) database.TargetRow {
		return database.TargetRow{
			Name:          name,
			Labels:        map[string]string{"region": "us"},
			Params:        json.RawMessage(`{"cluster": {"replicas": 3, "name": "` + name + `"}}`),
			TargetSetName: "fleet",
		}
	}
	// createTargets creates the named Targets in the Project and returns
	// their ids by name.
	createTargets := func(t *testing.T, project string, names ...string) map[string]uuid.UUID {
		t.Helper()
		ids := make(map[string]uuid.UUID, len(names))
		for _, name := range names {
			target, err := store.Create(ctx, project, newTarget(name))
			require.NoError(t, err)
			ids[name] = target.ID
		}
		return ids
	}
	// createStage records a Stage as the mirror would: a target-aware one
	// with selectors, a classic one without.
	createStage := func(t *testing.T, project, name string, targetAware bool) {
		t.Helper()
		var selectors []byte
		if targetAware {
			selectors = []byte(`[{}]`)
		}
		require.NoError(t, queries.UpsertStage(ctx, database.UpsertStageParams{
			ID:              project + "/" + name,
			ProjectName:     project,
			Name:            name,
			TargetSelectors: selectors,
			CreatedAt:       time.Now(),
		}))
	}
	// createRollout records a rollout of the Stage to the Targets, in the
	// given phase.
	createRollout := func(
		t *testing.T,
		project string,
		stageName string,
		name string,
		phase string,
		ids map[string]uuid.UUID,
		targets ...string,
	) uuid.UUID {
		t.Helper()
		targetIDs := make([]uuid.UUID, len(targets))
		for k, target := range targets {
			targetIDs[k] = ids[target]
		}
		created, err := queries.CreatePromotionRollout(ctx, database.CreatePromotionRolloutParams{
			ProjectName: project,
			StageName:   stageName,
			Name:        name,
			FreightName: "freight",
			TargetIds:   targetIDs,
		})
		require.NoError(t, err)
		_, err = queries.UpdatePromotionRolloutStatus(ctx, database.UpdatePromotionRolloutStatusParams{
			StatusPhase: phase,
			ProjectName: project,
			Name:        name,
		})
		require.NoError(t, err)
		return created.ID
	}
	// createPromotion records a child Promotion of the rollout to the
	// Target, as the mirror would; ids order Promotions by creation.
	createPromotionAt := func(
		t *testing.T,
		project string,
		id string,
		stageName string,
		rolloutID uuid.UUID,
		target string,
		phase string,
		createdAt time.Time,
	) {
		t.Helper()
		taken, err := queries.UpsertPromotion(ctx, database.UpsertPromotionParams{
			ID:                 id,
			ProjectName:        project,
			Name:               id,
			StageName:          stageName,
			FreightName:        "freight",
			TargetName:         &target,
			PromotionRolloutID: &rolloutID,
			Phase:              phase,
			CreatedAt:          createdAt.Truncate(time.Microsecond),
		})
		require.NoError(t, err)
		require.False(t, taken)
	}
	createPromotion := func(
		t *testing.T,
		project string,
		id string,
		stageName string,
		rolloutID uuid.UUID,
		target string,
		phase string,
	) {
		t.Helper()
		createPromotionAt(
			t, project, id, stageName, rolloutID, target, phase, time.Now(),
		)
	}
	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create returns the stored Target",
			run: func(t *testing.T) {
				target, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.Equal(t, "demo", target.ProjectName)
				require.Equal(t, "us-east", target.Name)
				require.NotEqual(t, uuid.Nil, target.ID)
				require.False(t, target.CreatedAt.IsZero())
				require.True(t, target.CreatedAt.Equal(target.UpdatedAt))
				require.Equal(t, map[string]string{"region": "us"}, target.Labels)
				require.JSONEq(t,
					`{"cluster": {"replicas": 3, "name": "us-east"}}`,
					string(target.Params),
				)
			},
		},
		{
			name: "the TargetSet name is stored and survives an update",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("managed"))
				require.NoError(t, err)
				require.Equal(t, "fleet", created.TargetSetName)
				updated, err := store.Update(
					ctx, "demo", database.TargetRow{Name: "managed", TargetSetName: "fleet"},
				)
				require.NoError(t, err)
				require.Equal(t, "fleet", updated.TargetSetName)
			},
		},
		{
			name: "an update must name the Target's own TargetSet",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("guarded"))
				require.NoError(t, err)
				other := newTarget("guarded")
				other.TargetSetName = "other"
				_, err = store.Update(ctx, "demo", other)
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, `managed by TargetSet "fleet"`)
				none := newTarget("guarded")
				none.TargetSetName = ""
				_, err = store.Update(ctx, "demo", none)
				require.ErrorIs(t, err, database.ErrInvalid)
				fetched, err := store.Get(ctx, "demo", "guarded")
				require.NoError(t, err)
				require.Equal(t, map[string]string{"region": "us"}, fetched.Labels)
			},
		},
		{
			name: "a Target without a TargetSet name is invalid",
			run: func(t *testing.T) {
				target := newTarget("unmanaged")
				target.TargetSetName = ""
				_, err := store.Create(ctx, "demo", target)
				require.ErrorIs(t, err, database.ErrInvalid)
				_, err = store.Get(ctx, "demo", "unmanaged")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "the schema itself requires a TargetSet name",
			run: func(t *testing.T) {
				_, err := queries.CreateTarget(ctx, database.CreateTargetParams{
					ProjectName: "demo", Name: "schema-check",
					Labels: []byte(`{}`), Params: []byte(`{}`),
				})
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "absent labels and params are stored as empty objects",
			run: func(t *testing.T) {
				target, err := store.Create(ctx, "demo", database.TargetRow{Name: "bare", TargetSetName: "fleet"})
				require.NoError(t, err)
				require.Empty(t, target.Labels)
				require.JSONEq(t, `{}`, string(target.Params))
				fetched, err := store.Get(ctx, "demo", "bare")
				require.NoError(t, err)
				require.Equal(t, target, fetched)
			},
		},
		{
			name: "names are unique within a Project only",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = store.Create(ctx, "demo", newTarget("us-east"))
				require.ErrorIs(t, err, database.ErrAlreadyExists)
				_, err = store.Create(ctx, "other", newTarget("us-east"))
				require.NoError(t, err)
			},
		},
		{
			name: "the database rejects labels that are not a string map",
			run: func(t *testing.T) {
				for _, bad := range []string{`[]`, `{"a": 1}`, `{"a": null}`} {
					_, err := pool.Exec(ctx,
						"INSERT INTO targets (project_name, name, target_set_name, labels) "+
							"VALUES ('demo', 'x', 'fleet', $1)",
						bad,
					)
					databasetest.RequirePGError(t, err, "23514")
				}
				_, err := pool.Exec(ctx,
					"INSERT INTO targets (project_name, name, target_set_name, params) "+
						"VALUES ('demo', 'x', 'fleet', '[]')",
				)
				databasetest.RequirePGError(t, err, "23514")
				_, err = pool.Exec(ctx,
					"INSERT INTO targets (project_name, name, target_set_name) VALUES ('demo', 'x', '')",
				)
				databasetest.RequirePGError(t, err, "23514")
			},
		},
		{
			name: "a value the database cannot store is invalid",
			run: func(t *testing.T) {
				target := newTarget("nul")
				target.Params = json.RawMessage(`{"cluster": "a\u0000b"}`)
				_, err := store.Create(ctx, "demo", target)
				require.ErrorIs(t, err, database.ErrInvalid)
			},
		},
		{
			name: "list is per Project and ordered by name",
			run: func(t *testing.T) {
				for _, name := range []string{"b", "c", "a"} {
					_, err := store.Create(ctx, "demo", newTarget(name))
					require.NoError(t, err)
				}
				_, err := store.Create(ctx, "other", newTarget("z"))
				require.NoError(t, err)
				targets, err := store.List(ctx, "demo")
				require.NoError(t, err)
				require.Equal(t, []string{"a", "b", "c"}, targetNames(targets))
				targets, err = store.List(ctx, "missing")
				require.NoError(t, err)
				require.Empty(t, targets)
			},
		},
		{
			name: "count totals a Project's Targets and those healthy by each Stage's latest rollout",
			run: func(t *testing.T) {
				const project = "health"
				ids := createTargets(t, project, "t1", "t2", "t3", "t4", "t5", "t6")
				_, err := store.Create(ctx, "elsewhere", newTarget("t1"))
				require.NoError(t, err)
				stage := func(name string, targetAware bool) {
					createStage(t, project, name, targetAware)
				}
				rollout := func(stageName, name, phase string, targets ...string) uuid.UUID {
					return createRollout(t, project, stageName, name, phase, ids, targets...)
				}
				promotion := func(id, stageName string, rolloutID uuid.UUID, target, phase string) {
					createPromotion(t, project, id, stageName, rolloutID, target, phase)
				}

				stage("a", true)
				stage("b", true)
				stage("classic", false)
				stage("idle", true)

				// a's latest rollout succeeded. t1 succeeded; t2 failed and its
				// retry succeeded; t3 has no Promotion and takes the rollout's
				// phase.
				a1 := rollout("a", "a.01", "Succeeded", "t1", "t2", "t3")
				promotion("p01", "a", a1, "t1", "Succeeded")
				promotion("p02", "a", a1, "t2", "Failed")
				promotion("p03", "a", a1, "t2", "Succeeded")

				// b once succeeded at t1, but its rollout in progress, which
				// stands for it, is still promoting there; t4 is Pending.
				b1 := rollout("b", "b.01", "Succeeded", "t1")
				promotion("p04", "b", b1, "t1", "Succeeded")
				b2 := rollout("b", "b.02", "Running", "t1", "t4")
				promotion("p05", "b", b2, "t1", "Running")

				// A classic Stage says nothing about Targets, even with a
				// rollout left behind that failed at t2.
				c1 := rollout("classic", "classic.01", "Errored", "t2")
				promotion("p06", "classic", c1, "t2", "Errored")

				// An Errored rollout fails its Targets with no Promotion.
				rollout("idle", "idle.01", "Errored", "t5")

				// t2 and t3 are healthy; t1 is promoting from b; t4 is
				// Pending; t5 failed; t6 was never promoted to.
				counts, err := store.Count(ctx, project)
				require.NoError(t, err)
				require.Equal(t, database.CountTargetsRow{Total: 6, Healthy: 2}, counts)

				counts, err = store.Count(ctx, "elsewhere")
				require.NoError(t, err)
				require.Equal(t, database.CountTargetsRow{Total: 1}, counts)
				counts, err = store.Count(ctx, "missing")
				require.NoError(t, err)
				require.Equal(t, database.CountTargetsRow{}, counts)
			},
		},
		{
			name: "count takes the newest Promotion to a Target by creation time, not ID",
			run: func(t *testing.T) {
				// Promotion IDs are ULIDs, whose timestamps have only
				// millisecond precision, so two Promotions created within the
				// same millisecond may order by their random bits. created_at
				// keeps microseconds and must decide which Promotion is newest.
				const project = "ordering"
				ids := createTargets(t, project, "t1")
				createStage(t, project, "s", true)
				r1 := createRollout(t, project, "s", "s.01", "Succeeded", ids, "t1")
				now := time.Now()
				createPromotionAt(t, project, "z-older", "s", r1, "t1", "Failed", now)
				createPromotionAt(
					t, project, "a-newer", "s", r1, "t1", "Succeeded",
					now.Add(time.Microsecond),
				)

				counts, err := store.Count(ctx, project)
				require.NoError(t, err)
				require.Equal(t, database.CountTargetsRow{Total: 1, Healthy: 1}, counts)
			},
		},
		{
			name: "count reads only the Project's own rows when Target and Stage names repeat",
			run: func(t *testing.T) {
				// Three Projects with the same Target and Stage names. Their
				// Promotions interleave, so the newest Promotion to a Target by
				// name is often another Project's.
				names := []string{"t1", "t2", "t3"}
				north := createTargets(t, "north", names...)
				south := createTargets(t, "south", names...)
				west := createTargets(t, "west", names...)
				createTargets(t, "east", names...)
				for _, project := range []string{"north", "south", "west", "east"} {
					createStage(t, project, "s", true)
				}

				// north promoted everywhere successfully.
				n1 := createRollout(t, "north", "s", "s.01", "Succeeded", north, names...)
				// south's rollout failed at t1, succeeded at t2, and has no
				// Promotion to t3, which takes the rollout's phase.
				s1 := createRollout(t, "south", "s", "s.01", "Failed", south, names...)
				// west's rollout is still running: t1 succeeded, t2 and t3 are
				// Pending.
				w1 := createRollout(t, "west", "s", "s.01", "Running", west, names...)
				// east has Targets but has never promoted to them.

				createPromotion(t, "north", "x01", "s", n1, "t1", "Succeeded")
				createPromotion(t, "west", "x02", "s", w1, "t1", "Succeeded")
				createPromotion(t, "south", "x03", "s", s1, "t1", "Failed")
				createPromotion(t, "north", "x04", "s", n1, "t2", "Succeeded")
				createPromotion(t, "south", "x05", "s", s1, "t2", "Succeeded")
				createPromotion(t, "north", "x06", "s", n1, "t3", "Succeeded")

				for project, want := range map[string]database.CountTargetsRow{
					"north": {Total: 3, Healthy: 3},
					"south": {Total: 3, Healthy: 1},
					"west":  {Total: 3, Healthy: 1},
					"east":  {Total: 3, Healthy: 0},
				} {
					counts, err := store.Count(ctx, project)
					require.NoError(t, err)
					require.Equal(t, want, counts, project)
				}
			},
		},
		{
			name: "get reports a missing Target",
			run: func(t *testing.T) {
				_, err := store.Get(ctx, "demo", "absent")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "updates keep identity and advance updated_at",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				first, err := store.Update(ctx, "demo", database.TargetRow{
					Name:          "us-east",
					Labels:        map[string]string{"region": "eu"},
					TargetSetName: "fleet",
				})
				require.NoError(t, err)
				second, err := store.Update(
					ctx, "demo", database.TargetRow{Name: "us-east", TargetSetName: "fleet"},
				)
				require.NoError(t, err)
				require.Equal(t, created.ID, second.ID)
				require.True(t, created.CreatedAt.Equal(second.CreatedAt))
				require.Equal(t, map[string]string{"region": "eu"}, first.Labels)
				require.Empty(t, second.Labels)
				require.JSONEq(t, `{}`, string(second.Params))
				require.True(t, first.UpdatedAt.After(created.UpdatedAt))
				require.True(t, second.UpdatedAt.After(first.UpdatedAt))
			},
		},
		{
			name: "updated_at advances even when the clock does not",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				_, err = pool.Exec(ctx, "UPDATE targets SET updated_at = '2999-01-01'")
				require.NoError(t, err)
				future, err := store.Get(ctx, "demo", "us-east")
				require.NoError(t, err)
				updated, err := store.Update(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.True(t, future.UpdatedAt.After(created.UpdatedAt))
				require.True(t, updated.UpdatedAt.After(future.UpdatedAt))
			},
		},
		{
			name: "update preconditions",
			run: func(t *testing.T) {
				created, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				stale := newTarget("us-east")
				stale.UpdatedAt = created.UpdatedAt.Add(-time.Microsecond)
				_, err = store.Update(ctx, "demo", stale)
				require.ErrorIs(t, err, database.ErrConflict)
				recreated := newTarget("us-east")
				recreated.ID = uuid.New()
				_, err = store.Update(ctx, "demo", recreated)
				require.ErrorIs(t, err, database.ErrConflict)
				current := newTarget("us-east")
				current.ID = created.ID
				current.UpdatedAt = created.UpdatedAt
				_, err = store.Update(ctx, "demo", current)
				require.NoError(t, err)
				_, err = store.Update(ctx, "demo", newTarget("absent"))
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "delete",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.NoError(t, store.Delete(ctx, "demo", "us-east"))
				require.ErrorIs(t, store.Delete(ctx, "demo", "us-east"), database.ErrNotFound)
				_, err = store.Get(ctx, "demo", "us-east")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "calls join the caller's transaction",
			run: func(t *testing.T) {
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = New(queries.WithTx(tx)).Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				require.NoError(t, tx.Rollback(ctx))
				_, err = store.Get(ctx, "demo", "us-east")
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "a locked row respects the caller's deadline",
			run: func(t *testing.T) {
				_, err := store.Create(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
				tx, err := pool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()
				_, err = tx.Exec(ctx, "SELECT id FROM targets FOR UPDATE")
				require.NoError(t, err)
				deadlineCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				_, err = store.Update(deadlineCtx, "demo", newTarget("us-east"))
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.NoError(t, tx.Rollback(ctx))
				_, err = store.Update(ctx, "demo", newTarget("us-east"))
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE targets CASCADE")
			require.NoError(t, err)
			testCase.run(t)
		})
	}
}

func targetNames(targets []database.TargetRow) []string {
	names := make([]string, len(targets))
	for i, target := range targets {
		names[i] = target.Name
	}
	return names
}
