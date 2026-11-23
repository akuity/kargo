//go:build integration && db

package targetstatusstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
)

// fixture writes what the mirror and the rollout endpoints would have
// written, one thing a minute, so that everything it records is ordered the
// way the calls that recorded it are.
type fixture struct {
	t       *testing.T
	pool    *pgxpool.Pool
	queries *database.Queries
	store   Store
	project string
	clock   time.Time
	seq     int
}

func newFixture(t *testing.T, pool *pgxpool.Pool, project string) *fixture {
	queries := database.New(pool)
	return &fixture{
		t:       t,
		pool:    pool,
		queries: queries,
		store:   New(queries),
		project: project,
		clock:   time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func (f *fixture) tick() time.Time {
	f.clock = f.clock.Add(time.Minute)
	f.seq++
	return f.clock
}

// stage records a target-aware Stage with the id.
func (f *fixture) stage(id, name string) {
	f.t.Helper()
	require.NoError(f.t, f.queries.UpsertStage(context.Background(), database.UpsertStageParams{
		ID:              id,
		ProjectName:     f.project,
		Name:            name,
		TargetSelectors: []byte(`[{}]`),
		CreatedAt:       f.tick(),
	}))
}

// target returns the id of the named Target, creating it if need be.
func (f *fixture) target(name string) uuid.UUID {
	f.t.Helper()
	row, err := f.queries.GetTarget(context.Background(), database.GetTargetParams{
		ProjectName: f.project,
		Name:        name,
	})
	if err != nil {
		row, err = f.queries.CreateTarget(context.Background(), database.CreateTargetParams{
			ProjectName:   f.project,
			Name:          name,
			Labels:        []byte(`{"region":"us"}`),
			Params:        []byte(`{}`),
			TargetSetName: "fleet",
		})
		require.NoError(f.t, err)
	}
	return row.ID
}

// rollout is a recorded PromotionRollout.
type rollout struct {
	id     uuid.UUID
	number int64
	stage  string
	name   string
}

// rollout records a PromotionRollout of the Stage to the Targets with the
// ids, in the given phase.
func (f *fixture) rollout(stageName, phase string, ids ...uuid.UUID) rollout {
	f.t.Helper()
	ctx := context.Background()
	created := f.tick()
	name := fmt.Sprintf("%s.rollout-%04d", stageName, f.seq)
	row, err := f.queries.CreatePromotionRollout(ctx, database.CreatePromotionRolloutParams{
		ProjectName: f.project,
		StageName:   stageName,
		Name:        name,
		FreightName: "freight",
		TargetIds:   ids,
	})
	require.NoError(f.t, err)
	// The database stamps a rollout with the real time; put it on the
	// fixture's clock instead, where the Promotions are.
	_, err = f.pool.Exec(ctx, "UPDATE promotion_rollouts SET created_at = $1 WHERE id = $2", created, row.ID)
	require.NoError(f.t, err)
	r := rollout{id: row.ID, number: row.Number, stage: stageName, name: name}
	f.phase(r, phase)
	return r
}

// phase moves a rollout to the phase.
func (f *fixture) phase(r rollout, phase string) {
	f.t.Helper()
	_, err := f.queries.UpdatePromotionRolloutStatus(
		context.Background(),
		database.UpdatePromotionRolloutStatusParams{
			StatusPhase: phase,
			ProjectName: f.project,
			Name:        r.name,
		},
	)
	require.NoError(f.t, err)
}

// promotion records a Promotion of the Stage to the named Target, in the
// given phase, as part of the rollout if one is given, and returns its name.
// It renders the named Freight, one piece from each of the origins "app" and
// "config" in that order; with one name, only "app".
func (f *fixture) promotion(stageName, target, phase string, r *rollout, freight ...string) string {
	f.t.Helper()
	created := f.tick()
	name := fmt.Sprintf("%s.promotion-%04d", stageName, f.seq)
	args := database.UpsertPromotionParams{
		ID:           fmt.Sprintf("%s-%06d", f.project, f.seq),
		ProjectName:  f.project,
		Name:         name,
		StageName:    stageName,
		TargetName:   &target,
		Phase:        phase,
		CreatedAt:    created,
		OriginKinds:  []string{},
		OriginNames:  []string{},
		FreightNames: []string{},
	}
	if len(freight) > 0 {
		args.FreightName = freight[0]
	}
	for i, freightName := range freight {
		args.OriginKinds = append(args.OriginKinds, "Warehouse")
		args.OriginNames = append(args.OriginNames, []string{"app", "config"}[i])
		args.FreightNames = append(args.FreightNames, freightName)
	}
	if r != nil {
		args.PromotionRolloutID = &r.id
	}
	if phase != "Pending" && phase != "Running" && phase != "" {
		finished := created.Add(30 * time.Second)
		args.FinishedAt = &finished
	}
	taken, err := f.queries.UpsertPromotion(context.Background(), args)
	require.NoError(f.t, err)
	require.False(f.t, taken)
	return name
}
