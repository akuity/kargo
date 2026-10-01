package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// targetNameConstraint is the unique constraint on a Target's name within its
// Project. Its violation is how the database reports a duplicate.
const targetNameConstraint = "targets_project_id_name_key"

// CreateTarget stores a new Target in the named Project and returns it as
// stored. The Project must already be mirrored; ErrProjectNotMirrored says
// it is not yet, ErrAlreadyExists that a Target of that name exists.
func (s *store) CreateTarget(
	ctx context.Context,
	project string,
	target *kargoapi.Target,
) (*kargoapi.Target, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	lbls, params, err := targetColumns(target)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.CreateTarget(ctx, CreateTargetParams{
		ProjectName: project,
		Name:        target.Name,
		Labels:      lbls,
		Params:      params,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrProjectNotMirrored
		}
		return nil, targetError(err, project, target.Name)
	}
	return targetFromRow(row, project)
}

// GetTarget returns the named Target, or ErrNotFound.
func (s *store) GetTarget(
	ctx context.Context,
	project string,
	name string,
) (*kargoapi.Target, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	row, err := s.queries.GetTarget(ctx, GetTargetParams{
		ProjectName: project,
		Name:        name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, targetError(err, project, name)
	}
	return targetFromRow(row, project)
}

// ListTargets returns every Target in the named Project, ordered by name. A
// Project with no Targets, or none at all, yields an empty list.
func (s *store) ListTargets(
	ctx context.Context,
	project string,
) ([]kargoapi.Target, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	rows, err := s.queries.ListTargets(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("error listing Targets in Project %q: %w", project, err)
	}
	targets := make([]kargoapi.Target, len(rows))
	for i, row := range rows {
		target, err := targetFromRow(row, project)
		if err != nil {
			return nil, err
		}
		targets[i] = *target
	}
	return targets, nil
}

// UpdateTarget replaces the labels and params of the Target named by the
// argument and returns it as stored. A resource version or UID set on the
// argument is a precondition: ErrConflict says the row no longer matches it.
// Without either, the update is unconditional.
func (s *store) UpdateTarget(
	ctx context.Context,
	project string,
	target *kargoapi.Target,
) (*kargoapi.Target, error) {
	lbls, params, err := targetColumns(target)
	if err != nil {
		return nil, err
	}
	var row TargetRow
	err = s.transact(ctx, func(txCtx context.Context, queries *Queries) error {
		current, txErr := queries.GetTargetForUpdate(txCtx, GetTargetForUpdateParams{
			ProjectName: project,
			Name:        target.Name,
		})
		if txErr != nil {
			if errors.Is(txErr, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return txErr
		}
		if txErr = checkTargetPreconditions(target, current); txErr != nil {
			return txErr
		}
		row, txErr = queries.UpdateTarget(txCtx, UpdateTargetParams{
			ID:     current.ID,
			Labels: lbls,
			Params: params,
		})
		return txErr
	})
	if err != nil {
		return nil, targetError(err, project, target.Name)
	}
	return targetFromRow(row, project)
}

// DeleteTarget removes the named Target, or returns ErrNotFound.
func (s *store) DeleteTarget(ctx context.Context, project string, name string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	deleted, err := s.queries.DeleteTarget(ctx, DeleteTargetParams{
		ProjectName: project,
		Name:        name,
	})
	if err == nil && deleted == 0 {
		err = ErrNotFound
	}
	if err != nil {
		return targetError(err, project, name)
	}
	return nil
}

// checkTargetPreconditions compares the identity and version a caller
// asserts, if any, against the row about to be written.
func checkTargetPreconditions(target *kargoapi.Target, current TargetRow) error {
	if target.UID != "" && string(target.UID) != current.ID.String() {
		return fmt.Errorf("%w: the Target was deleted and recreated", ErrConflict)
	}
	if target.ResourceVersion != "" &&
		target.ResourceVersion != resourceVersion(current.UpdatedAt) {
		return fmt.Errorf("%w: the Target has been modified", ErrConflict)
	}
	return nil
}

// targetError names the Target an error concerns and translates what the
// database reports into the package's sentinel errors, so that callers can
// tell a duplicate, a missing Project, or a bad value apart from a failure.
func targetError(err error, project, name string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == targetNameConstraint:
			err = ErrAlreadyExists
		case pgErr.Code == "23503":
			// The Project row went away between resolving it and the insert.
			err = ErrProjectNotMirrored
		case pgErr.Code == "23514" || pgErr.Code[:2] == "22":
			err = fmt.Errorf("%w: %s", ErrInvalid, pgErr.Message)
		}
	}
	// nolint:staticcheck
	return fmt.Errorf("Target %q in Project %q: %w", name, project, err)
}

// targetColumns encodes the parts of a Target that the database stores. Nil
// maps become empty objects: the columns are NOT NULL and a JSON null would
// fail their CHECK constraints.
func targetColumns(target *kargoapi.Target) (lbls, params []byte, err error) {
	labelMap := target.Labels
	if labelMap == nil {
		labelMap = map[string]string{}
	}
	if lbls, err = json.Marshal(labelMap); err != nil {
		return nil, nil, fmt.Errorf(
			"error encoding labels of Target %q: %w", target.Name, err,
		)
	}
	paramMap := make(map[string]json.RawMessage, len(target.Spec.Params))
	for key, value := range target.Spec.Params {
		paramMap[key] = json.RawMessage(value.Raw)
	}
	if params, err = json.Marshal(paramMap); err != nil {
		return nil, nil, fmt.Errorf(
			"error encoding params of Target %q: %w", target.Name, err,
		)
	}
	return lbls, params, nil
}

// targetFromRow presents a row in the shape of the Target resource, so that
// code written against the resource consumes rows unchanged. The row's id is
// the UID and its last update the resource version.
func targetFromRow(row TargetRow, project string) (*kargoapi.Target, error) {
	var rawParams map[string]json.RawMessage
	if err := json.Unmarshal(row.Params, &rawParams); err != nil {
		return nil, fmt.Errorf("error decoding params of Target %q: %w", row.Name, err)
	}
	var params map[string]apiextensionsv1.JSON
	if len(rawParams) > 0 {
		params = make(map[string]apiextensionsv1.JSON, len(rawParams))
		for key, raw := range rawParams {
			params[key] = apiextensionsv1.JSON{Raw: raw}
		}
	}
	var lbls map[string]string
	if len(row.Labels) > 0 {
		lbls = map[string]string(row.Labels)
	}
	return &kargoapi.Target{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Target",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         project,
			Name:              row.Name,
			UID:               types.UID(row.ID.String()),
			ResourceVersion:   resourceVersion(row.UpdatedAt),
			CreationTimestamp: metav1.NewTime(row.CreatedAt),
			Labels:            lbls,
		},
		Spec: kargoapi.TargetSpec{Params: params},
	}, nil
}

// resourceVersion derives a Target's resource version from its row's last
// update, which the schema guarantees grows with every write to the row.
func resourceVersion(updatedAt time.Time) string {
	return strconv.FormatInt(updatedAt.UnixMicro(), 10)
}
