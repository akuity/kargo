// Package promotionrolloutstore keeps PromotionRollouts, which are authored
// in the database rather than mirrored from Kubernetes.
package promotionrolloutstore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/akuity/kargo/pkg/database"
)

// nameConstraint is the unique constraint on a PromotionRollout's name
// within its Project. Its violation is how the database reports a duplicate.
const nameConstraint = "promotion_rollouts_project_name_name_key"

// Key identifies a PromotionRollout: the Project it belongs to and its name.
// It is what everything outside the database addresses a rollout by, and
// the request type of the rollout controller.
type Key struct {
	Project string
	Name    string
}

func (k Key) String() string { return k.Project + "/" + k.Name }

// Rollout is a PromotionRollout as stored: its row and its Targets, ordered
// by Target name. Targets is empty until ResolveTargets has run, which
// TargetsResolvedAt tells from a rollout that resolved to none. The
// database records which Targets a rollout promotes to and nothing about
// the Promotions that do so; those say so themselves.
type Rollout struct {
	database.PromotionRolloutRow
	Targets []database.ListPromotionRolloutTargetsRow
}

// Store reads and writes PromotionRollouts. Rollouts are addressed by Project
// name and rollout name.
type Store interface {
	Create(context.Context, string, database.PromotionRolloutRow) (Rollout, error)
	ResolveTargets(context.Context, Rollout, []uuid.UUID) (Rollout, error)
	Get(context.Context, string, string) (Rollout, error)
	GetMany(context.Context, string, []string) ([]Rollout, error)
	List(context.Context, string, string) ([]Rollout, error)
	ListOpen(context.Context) ([]Key, error)
	UpdateStatus(context.Context, string, database.PromotionRolloutRow) (database.PromotionRolloutRow, error)
}

type store struct {
	queries *database.Queries
}

// New returns a Store that runs its statements through queries. No call opens
// a transaction of its own, so a caller may bind queries to one, through
// Queries.WithTx, to make several calls atomic.
func New(queries *database.Queries) Store {
	return &store{queries: queries}
}

// Create stores a new PromotionRollout in the named Project and returns it
// as stored. Only the argument's name, Stage, Freight, max concurrency and
// creator are read; the database assigns the rest, including the rollout's
// number within its Stage, which it takes from the Stage's mirrored row.
// The Project, Stage and Freight are not checked: the caller confirms they
// exist. The rollout promotes to no Targets until ResolveTargets records
// them. database.ErrAlreadyExists says a rollout of that name exists, and
// database.ErrNotMirrored that the Stage has no mirrored row yet.
func (s *store) Create(
	ctx context.Context,
	project string,
	rollout database.PromotionRolloutRow,
) (Rollout, error) {
	created, err := s.queries.CreatePromotionRollout(ctx, database.CreatePromotionRolloutParams{
		ProjectName:   project,
		Name:          rollout.Name,
		StageName:     rollout.StageName,
		FreightName:   rollout.FreightName,
		MaxConcurrent: rollout.MaxConcurrent,
		CreatedBy:     rollout.CreatedBy,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// nolint:staticcheck
		err = fmt.Errorf("Stage %q: %w", rollout.StageName, database.ErrNotMirrored)
	}
	if err != nil {
		return Rollout{}, rolloutError(err, project, rollout.Name)
	}
	return Rollout{
		PromotionRolloutRow: database.PromotionRolloutRow(created),
		Targets:             []database.ListPromotionRolloutTargetsRow{},
	}, nil
}

// ResolveTargets records the Targets, by id, that the rollout promotes to
// and marks it resolved, and returns it as stored. It writes exactly once:
// a rollout already resolved is left alone and reported as
// database.ErrConflict, as is a Target id that no longer exists, in which
// case nothing is written. No ids resolve the rollout to no Targets.
func (s *store) ResolveTargets(
	ctx context.Context,
	rollout Rollout,
	targetIDs []uuid.UUID,
) (Rollout, error) {
	ids := slices.Clone(targetIDs)
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	ids = slices.Compact(ids)
	if ids == nil {
		// A NULL array would insert nothing either, but is not what the
		// column's cast expects.
		ids = []uuid.UUID{}
	}
	resolved, err := s.queries.SetPromotionRolloutTargets(ctx, database.SetPromotionRolloutTargetsParams{
		ID:        rollout.ID,
		TargetIds: ids,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		err = fmt.Errorf("%w: the PromotionRollout's Targets are already resolved", database.ErrConflict)
	}
	if err != nil {
		return Rollout{}, rolloutError(err, rollout.ProjectName, rollout.Name)
	}
	rollouts, err := s.attachTargets(ctx, []database.PromotionRolloutRow{database.PromotionRolloutRow(resolved)})
	if err != nil {
		return Rollout{}, rolloutError(err, rollout.ProjectName, rollout.Name)
	}
	return rollouts[0], nil
}

// Get returns the named PromotionRollout, or database.ErrNotFound.
func (s *store) Get(ctx context.Context, project, name string) (Rollout, error) {
	row, err := s.queries.GetPromotionRollout(ctx, database.GetPromotionRolloutParams{
		ProjectName: project,
		Name:        name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = database.ErrNotFound
		}
		return Rollout{}, rolloutError(err, project, name)
	}
	rollouts, err := s.attachTargets(ctx, []database.PromotionRolloutRow{row})
	if err != nil {
		return Rollout{}, rolloutError(err, project, name)
	}
	return rollouts[0], nil
}

// GetMany returns those of the named PromotionRollouts in the Project that
// exist, ordered by name. No names yield no rollouts and no query.
func (s *store) GetMany(ctx context.Context, project string, names []string) ([]Rollout, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := s.queries.ListPromotionRolloutsByName(ctx, database.ListPromotionRolloutsByNameParams{
		ProjectName: project,
		Names:       names,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting PromotionRollouts in Project %q: %w", project, err)
	}
	rollouts, err := s.attachTargets(ctx, rows)
	if err != nil {
		return nil, fmt.Errorf("error getting PromotionRollouts in Project %q: %w", project, err)
	}
	return rollouts, nil
}

// List returns every PromotionRollout in the named Project, or only those of
// the named Stage when stage is not empty, in creation order. A Project with
// no rollouts, or none at all, yields an empty list.
func (s *store) List(ctx context.Context, project, stage string) ([]Rollout, error) {
	args := database.ListPromotionRolloutsParams{ProjectName: project}
	if stage != "" {
		args.StageName = &stage
	}
	rows, err := s.queries.ListPromotionRollouts(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("error listing PromotionRollouts in Project %q: %w", project, err)
	}
	rollouts, err := s.attachTargets(ctx, rows)
	if err != nil {
		return nil, fmt.Errorf("error listing PromotionRollouts in Project %q: %w", project, err)
	}
	return rollouts, nil
}

// ListOpen returns the key of every PromotionRollout, in any Project, that
// has not reached a terminal phase.
func (s *store) ListOpen(ctx context.Context) ([]Key, error) {
	rows, err := s.queries.ListOpenPromotionRollouts(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing open PromotionRollouts: %w", err)
	}
	keys := make([]Key, len(rows))
	for i, row := range rows {
		keys[i] = Key{Project: row.ProjectName, Name: row.Name}
	}
	return keys, nil
}

// UpdateStatus replaces the status of the PromotionRollout named by the
// argument -- its phase, reason, message and timestamps -- and returns the
// row as stored. A non-zero ID or UpdatedAt on the argument is a
// precondition: database.ErrConflict says the row no longer matches it.
// Without either, the update is unconditional.
func (s *store) UpdateStatus(
	ctx context.Context,
	project string,
	rollout database.PromotionRolloutRow,
) (database.PromotionRolloutRow, error) {
	args := database.UpdatePromotionRolloutStatusParams{
		ProjectName:      project,
		Name:             rollout.Name,
		StatusPhase:      rollout.StatusPhase,
		StatusReason:     rollout.StatusReason,
		StatusMessage:    rollout.StatusMessage,
		StatusStartedAt:  rollout.StatusStartedAt,
		StatusFinishedAt: rollout.StatusFinishedAt,
	}
	if rollout.ID != uuid.Nil {
		args.ID = &rollout.ID
	}
	if !rollout.UpdatedAt.IsZero() {
		args.UpdatedAt = &rollout.UpdatedAt
	}
	row, err := s.queries.UpdatePromotionRolloutStatus(ctx, args)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.updateMissed(ctx, project, rollout)
	}
	if err != nil {
		return database.PromotionRolloutRow{}, rolloutError(err, project, rollout.Name)
	}
	return row, nil
}

// updateMissed explains an update that matched no row: either the rollout
// does not exist, or it no longer satisfies the update's preconditions.
func (s *store) updateMissed(
	ctx context.Context,
	project string,
	rollout database.PromotionRolloutRow,
) error {
	current, err := s.queries.GetPromotionRollout(ctx, database.GetPromotionRolloutParams{
		ProjectName: project,
		Name:        rollout.Name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err = checkPreconditions(rollout, current); err != nil {
		return err
	}
	// The row changed again between the update and this read.
	return fmt.Errorf("%w: the PromotionRollout has been modified", database.ErrConflict)
}

// attachTargets pairs each row with its Targets, read in one query for all
// of them, keeping the rows' order. A rollout with no Targets gets an empty
// list rather than none.
func (s *store) attachTargets(
	ctx context.Context,
	rows []database.PromotionRolloutRow,
) ([]Rollout, error) {
	rollouts := make([]Rollout, len(rows))
	if len(rows) == 0 {
		return rollouts, nil
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		rollouts[i] = Rollout{
			PromotionRolloutRow: row,
			Targets:             []database.ListPromotionRolloutTargetsRow{},
		}
	}
	targets, err := s.queries.ListPromotionRolloutTargets(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets of PromotionRollouts: %w", err)
	}
	return groupTargets(rollouts, targets), nil
}

// groupTargets appends each Target row to the rollout it belongs to. Rows of
// a rollout that is not among the rollouts are dropped.
func groupTargets(
	rollouts []Rollout,
	targets []database.ListPromotionRolloutTargetsRow,
) []Rollout {
	byID := make(map[uuid.UUID]int, len(rollouts))
	for i, rollout := range rollouts {
		byID[rollout.ID] = i
	}
	for _, target := range targets {
		if i, ok := byID[target.PromotionRolloutID]; ok {
			rollouts[i].Targets = append(rollouts[i].Targets, target)
		}
	}
	return rollouts
}

// checkPreconditions compares the identity and version a caller asserts, if
// any, against the row about to be written. UpdatedAt is the version: the
// schema advances it with every write to the row.
func checkPreconditions(rollout, current database.PromotionRolloutRow) error {
	if rollout.ID != uuid.Nil && rollout.ID != current.ID {
		return fmt.Errorf("%w: the PromotionRollout was deleted and recreated", database.ErrConflict)
	}
	if !rollout.UpdatedAt.IsZero() && !rollout.UpdatedAt.Equal(current.UpdatedAt) {
		return fmt.Errorf("%w: the PromotionRollout has been modified", database.ErrConflict)
	}
	return nil
}

// rolloutError names the PromotionRollout an error concerns and translates
// what the database reports into the sentinel errors of pkg/database, so
// that callers can tell a duplicate, a vanished Target or a bad value apart
// from a failure.
func rolloutError(err error, project, name string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == nameConstraint:
			err = database.ErrAlreadyExists
		case pgErr.Code == "23503":
			// A Target the rollout is being resolved to no longer exists.
			err = fmt.Errorf("%w: %s", database.ErrConflict, pgErr.Message)
		case pgErr.Code == "23514" || strings.HasPrefix(pgErr.Code, "22"):
			err = fmt.Errorf("%w: %s", database.ErrInvalid, pgErr.Message)
		}
	}
	// nolint:staticcheck
	return fmt.Errorf("PromotionRollout %q in Project %q: %w", name, project, err)
}
