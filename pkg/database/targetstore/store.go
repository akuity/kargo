// Package targetstore keeps Targets, which are authored in the database
// rather than mirrored from Kubernetes.
package targetstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/akuity/kargo/pkg/database"
)

// nameConstraint is the unique constraint on a Target's name within its
// Project. Its violation is how the database reports a duplicate.
const nameConstraint = "targets_project_name_name_key"

// Store reads and writes Targets. Targets are addressed by Project name and
// Target name.
type Store interface {
	Create(context.Context, string, database.TargetRow) (database.TargetRow, error)
	Get(context.Context, string, string) (database.TargetRow, error)
	List(context.Context, string) ([]database.TargetRow, error)
	Update(context.Context, string, database.TargetRow) (database.TargetRow, error)
	Delete(context.Context, string, string) error
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

// Create stores a new Target in the named Project and returns it as stored.
// Only the argument's name, labels and params are read; the database assigns
// the rest. The Project is not checked: the caller confirms it exists.
// database.ErrAlreadyExists says a Target of that name exists.
func (s *store) Create(
	ctx context.Context,
	project string,
	target database.TargetRow,
) (database.TargetRow, error) {
	lbls, params, err := columns(target)
	if err != nil {
		return database.TargetRow{}, err
	}
	row, err := s.queries.CreateTarget(ctx, database.CreateTargetParams{
		ProjectName: project,
		Name:        target.Name,
		Labels:      lbls,
		Params:      params,
	})
	if err != nil {
		return database.TargetRow{}, targetError(err, project, target.Name)
	}
	return row, nil
}

// Get returns the named Target, or database.ErrNotFound.
func (s *store) Get(
	ctx context.Context,
	project string,
	name string,
) (database.TargetRow, error) {
	row, err := s.queries.GetTarget(ctx, database.GetTargetParams{
		ProjectName: project,
		Name:        name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = database.ErrNotFound
		}
		return database.TargetRow{}, targetError(err, project, name)
	}
	return row, nil
}

// List returns every Target in the named Project, ordered by name. A Project
// with no Targets, or none at all, yields an empty list.
func (s *store) List(
	ctx context.Context,
	project string,
) ([]database.TargetRow, error) {
	rows, err := s.queries.ListTargets(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", project, err)
	}
	return rows, nil
}

// Update replaces the labels and params of the Target named by the argument
// and returns it as stored. A non-zero ID or UpdatedAt on the argument is a
// precondition: database.ErrConflict says the row no longer matches it.
// Without either, the update is unconditional.
func (s *store) Update(
	ctx context.Context,
	project string,
	target database.TargetRow,
) (database.TargetRow, error) {
	lbls, params, err := columns(target)
	if err != nil {
		return database.TargetRow{}, err
	}
	args := database.UpdateTargetParams{
		ProjectName: project,
		Name:        target.Name,
		Labels:      lbls,
		Params:      params,
	}
	if target.ID != uuid.Nil {
		args.ID = &target.ID
	}
	if !target.UpdatedAt.IsZero() {
		args.UpdatedAt = &target.UpdatedAt
	}
	row, err := s.queries.UpdateTarget(ctx, args)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.updateMissed(ctx, project, target)
	}
	if err != nil {
		return database.TargetRow{}, targetError(err, project, target.Name)
	}
	return row, nil
}

// updateMissed explains an update that matched no row: either the Target
// does not exist, or it no longer satisfies the update's preconditions.
func (s *store) updateMissed(
	ctx context.Context,
	project string,
	target database.TargetRow,
) error {
	current, err := s.queries.GetTarget(ctx, database.GetTargetParams{
		ProjectName: project,
		Name:        target.Name,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return database.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err = checkPreconditions(target, current); err != nil {
		return err
	}
	// The row changed again between the update and this read.
	return fmt.Errorf("%w: the Target has been modified", database.ErrConflict)
}

// Delete removes the named Target, or returns database.ErrNotFound.
func (s *store) Delete(ctx context.Context, project string, name string) error {
	deleted, err := s.queries.DeleteTarget(ctx, database.DeleteTargetParams{
		ProjectName: project,
		Name:        name,
	})
	if err == nil && deleted == 0 {
		err = database.ErrNotFound
	}
	if err != nil {
		return targetError(err, project, name)
	}
	return nil
}

// checkPreconditions compares the identity and version a caller asserts, if
// any, against the row about to be written. UpdatedAt is the version: the
// schema advances it with every write to the row.
func checkPreconditions(target, current database.TargetRow) error {
	if target.ID != uuid.Nil && target.ID != current.ID {
		return fmt.Errorf("%w: the Target was deleted and recreated", database.ErrConflict)
	}
	if !target.UpdatedAt.IsZero() && !target.UpdatedAt.Equal(current.UpdatedAt) {
		return fmt.Errorf("%w: the Target has been modified", database.ErrConflict)
	}
	return nil
}

// targetError names the Target an error concerns and translates what the
// database reports into the sentinel errors of pkg/database, so that callers
// can tell a duplicate or a bad value apart from a failure.
func targetError(err error, project, name string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == nameConstraint:
			err = database.ErrAlreadyExists
		case pgErr.Code == "23514" || strings.HasPrefix(pgErr.Code, "22"):
			err = fmt.Errorf("%w: %s", database.ErrInvalid, pgErr.Message)
		}
	}
	// nolint:staticcheck
	return fmt.Errorf("Target %q in Project %q: %w", name, project, err)
}

// columns encodes the labels and params of a Target for the database. Absent
// ones become empty objects: the columns are NOT NULL and a JSON null would
// fail their CHECK constraints. Params are passed through as written; the
// database rejects any that are not a JSON object.
func columns(target database.TargetRow) (lbls, params []byte, err error) {
	labelMap := target.Labels
	if labelMap == nil {
		labelMap = map[string]string{}
	}
	if lbls, err = json.Marshal(labelMap); err != nil {
		return nil, nil, fmt.Errorf(
			"error encoding labels of Target %q: %w", target.Name, err,
		)
	}
	params = target.Params
	if len(params) == 0 {
		params = []byte(`{}`)
	}
	return lbls, params, nil
}
