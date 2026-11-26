// Package projectconfigstore keeps ProjectConfigs, which are authored in the
// database rather than in Kubernetes.
package projectconfigstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// nameConstraint is the unique constraint on a ProjectConfig's Project name.
// Its violation is how the database reports a duplicate.
const nameConstraint = "project_configs_project_name_key"

// pathConstraint is the unique constraint on a webhook receiver's path. Its
// violation is how the database reports a path another ProjectConfig owns.
const pathConstraint = "project_config_webhook_receivers_path_key"

// Config is a ProjectConfig as stored: its row and its receiver rows.
type Config struct {
	database.ProjectConfigRow
	Receivers []database.ProjectConfigWebhookReceiverRow
}

// Status is the outcome of a reconcile of a ProjectConfig: the generation
// and refresh token it observed, and what it found.
type Status struct {
	// ObservedID is the ID of the row the reconcile read. A row that has
	// since been replaced by another of the same name does not take the
	// status.
	ObservedID uuid.UUID
	// ObservedGeneration is the generation the reconcile read.
	ObservedGeneration int64
	// ObservedRefreshToken is the refresh token the reconcile read; "" when
	// the row had none.
	ObservedRefreshToken string
	Conditions           []metav1.Condition
	WebhookReceivers     []kargoapi.WebhookReceiverDetails
}

// Store reads and writes ProjectConfigs. A ProjectConfig is addressed by the
// name of its Project, and belongs to the Project with the UID it was
// created for: a row whose Project UID is not the caller's belongs to an
// earlier Project of the same name, and is treated as absent until
// AdoptForProject rebinds it to the current one.
//
// Every committed write is announced on Events. A failure to publish is
// logged, not returned.
type Store interface {
	// Get returns the Project's ProjectConfig, or database.ErrNotFound.
	Get(ctx context.Context, project, projectUID string) (Config, error)
	// GetUnfiltered returns the row for the Project name whatever its
	// project_uid. Only the reconciler's leftover check uses it.
	GetUnfiltered(ctx context.Context, project string) (Config, error)
	// List returns every ProjectConfig except those the mirrored projects
	// table shows to be leftovers, ordered by Project name.
	List(ctx context.Context) ([]Config, error)
	// Create stores a new ProjectConfig for the Project with the given UID,
	// replacing a leftover row of a deleted Project of the same name.
	Create(ctx context.Context, cfg *kargoapi.ProjectConfig, projectUID string) (Config, error)
	// Update replaces a ProjectConfig's labels, annotations and spec.
	Update(ctx context.Context, cfg *kargoapi.ProjectConfig, projectUID string) (Config, error)
	// Delete removes the Project's ProjectConfig, or returns
	// database.ErrNotFound.
	Delete(ctx context.Context, project, projectUID string) error
	// DeleteForProject removes the ProjectConfig of the deleted Project with
	// the given UID, if any.
	DeleteForProject(ctx context.Context, project, projectUID string) error
	// AdoptForProject rebinds the Project's ProjectConfig, if it belongs to
	// an earlier Project of the same name, to the Project with newUID. It
	// keeps the spec, status and webhook receivers, publishes Updated, and
	// reports whether a row was adopted. No row, or a row already bound to
	// newUID, is a no-op.
	AdoptForProject(ctx context.Context, project, newUID string) (adopted bool, err error)
	// UpdateStatus records the outcome of a reconcile and returns the names
	// of the webhook receivers whose paths another ProjectConfig owns.
	UpdateStatus(ctx context.Context, project string, status Status) (skipped []string, err error)
	// RequestRefresh asks for the ProjectConfig to be reconciled and returns
	// a token its status reports once it has been.
	RequestRefresh(ctx context.Context, project, projectUID string) (token string, err error)
	// GetByWebhookReceiverPath returns the ProjectConfig that owns a webhook
	// receiver path, or database.ErrNotFound.
	GetByWebhookReceiverPath(ctx context.Context, path string) (Config, error)
	// Import inserts a config with status copied from a CR, never
	// overwriting. Used by the importer only.
	Import(ctx context.Context, cfg *kargoapi.ProjectConfig, projectUID string) (inserted bool, err error)
}

type store struct {
	pool    *pgxpool.Pool
	queries *database.Queries
	pub     nats.Publisher
}

// New returns a Store that reads and writes through pool and announces its
// writes through pub. A nil pub announces nothing. Writes that take more than
// one statement run in transactions of their own, so the store, unlike
// targetstore, is not given queries to bind to a caller's transaction.
func New(pool *pgxpool.Pool, pub nats.Publisher) Store {
	return &store{
		pool:    pool,
		queries: database.New(pool),
		pub:     pub,
	}
}

// Get returns the Project's ProjectConfig. A row created for another Project
// UID is a leftover and is reported as database.ErrNotFound.
func (s *store) Get(ctx context.Context, project, projectUID string) (Config, error) {
	row, err := s.getRow(ctx, s.queries, project, projectUID)
	if err != nil {
		return Config{}, configError(err, project)
	}
	cfg, err := s.withReceivers(ctx, row)
	if err != nil {
		return Config{}, configError(err, project)
	}
	return cfg, nil
}

// GetUnfiltered returns the row for the Project name whatever Project UID it
// was created for, or database.ErrNotFound.
func (s *store) GetUnfiltered(ctx context.Context, project string) (Config, error) {
	row, err := s.queries.GetProjectConfig(ctx, project)
	if errors.Is(err, pgx.ErrNoRows) {
		err = database.ErrNotFound
	}
	if err != nil {
		return Config{}, configError(err, project)
	}
	cfg, err := s.withReceivers(ctx, row)
	if err != nil {
		return Config{}, configError(err, project)
	}
	return cfg, nil
}

// List returns every ProjectConfig, ordered by Project name, except those
// whose Project the mirror records under another UID. A ProjectConfig whose
// Project is not mirrored yet is listed. The receivers of all of them are
// read in one query.
func (s *store) List(ctx context.Context) ([]Config, error) {
	rows, err := s.queries.ListProjectConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing ProjectConfigs: %w", err)
	}
	configs := make([]Config, len(rows))
	if len(rows) == 0 {
		return configs, nil
	}
	ids := make([]uuid.UUID, len(rows))
	byID := make(map[uuid.UUID]int, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		byID[row.ID] = i
		configs[i] = Config{ProjectConfigRow: row}
	}
	receivers, err := s.queries.ListProjectConfigWebhookReceiversByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("error listing webhook receivers of ProjectConfigs: %w", err)
	}
	for _, receiver := range receivers {
		if i, ok := byID[receiver.ProjectConfigID]; ok {
			configs[i].Receivers = append(configs[i].Receivers, receiver)
		}
	}
	return configs, nil
}

// Create stores a new ProjectConfig for the Project with the given UID and
// returns it as stored. Its labels, annotations and spec are read as
// specColumns describes; the database assigns the rest. The Project is not
// checked: the caller confirms it exists.
//
// database.ErrAlreadyExists says the Project has a ProjectConfig. A row
// created for another UID is a leftover from a deleted Project of the same
// name, and is replaced in the same transaction: its receivers go with it,
// freeing their paths, and Deleted is published for it before Created is
// published for the new row.
func (s *store) Create(
	ctx context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (Config, error) {
	project, err := projectOf(cfg, projectUID)
	if err != nil {
		return Config{}, configError(err, project)
	}
	labels, annotations, spec, err := specColumns(cfg)
	if err != nil {
		return Config{}, configError(err, project)
	}
	var created database.ProjectConfigRow
	var replaced *database.ProjectConfigRow
	err = s.inTx(ctx, func(q *database.Queries) error {
		existing, txErr := q.GetProjectConfig(ctx, project)
		switch {
		case errors.Is(txErr, pgx.ErrNoRows):
		case txErr != nil:
			return txErr
		case existing.ProjectUID == projectUID:
			return database.ErrAlreadyExists
		default:
			deleted, delErr := q.DeleteProjectConfigByID(ctx, existing.ID)
			switch {
			case errors.Is(delErr, pgx.ErrNoRows):
				// Someone else removed it first; the insert decides who wins.
			case delErr != nil:
				return delErr
			default:
				replaced = &deleted
			}
		}
		created, txErr = q.InsertProjectConfig(ctx, database.InsertProjectConfigParams{
			ProjectName: project,
			ProjectUID:  projectUID,
			Labels:      labels,
			Annotations: annotations,
			Spec:        spec,
		})
		return txErr
	})
	if err != nil {
		return Config{}, configError(err, project)
	}
	if replaced != nil {
		s.publishDeleted(ctx, *replaced)
	}
	s.publishCreated(ctx, created)
	return Config{ProjectConfigRow: created}, nil
}

// Update replaces the labels, annotations and spec of the Project's
// ProjectConfig and returns it as stored. The generation is bumped only if
// the spec changed, which the database decides, since JSON equality there
// ignores key order. A row created for another Project UID is reported as
// database.ErrNotFound.
//
// A UID or resource version on the argument is a precondition, and
// database.ErrConflict says the row no longer matches it. A resource version
// that this store did not issue is database.ErrInvalid.
//
// The Updated event carries the row as read just before the update as its
// old row; a write that lands between the two is folded into it.
func (s *store) Update(
	ctx context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (Config, error) {
	project, err := projectOf(cfg, projectUID)
	if err != nil {
		return Config{}, configError(err, project)
	}
	labels, annotations, spec, err := specColumns(cfg)
	if err != nil {
		return Config{}, configError(err, project)
	}
	preconditions, err := preconditionsOf(cfg)
	if err != nil {
		return Config{}, configError(err, project)
	}
	old, err := s.getRow(ctx, s.queries, project, projectUID)
	if err != nil {
		return Config{}, configError(err, project)
	}
	args := database.UpdateProjectConfigSpecParams{
		ProjectName: project,
		Labels:      labels,
		Annotations: annotations,
		Spec:        spec,
		ID:          preconditions.ID,
		UpdatedAt:   preconditions.UpdatedAt,
	}
	if args.ID == nil {
		// Pin the update to the row whose Project UID was just checked, so
		// that a replacement made since is not updated in its place.
		args.ID = &old.ID
	}
	updated, err := s.queries.UpdateProjectConfigSpec(ctx, args)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.updateMissed(ctx, project, projectUID, preconditions)
	}
	if err != nil {
		return Config{}, configError(err, project)
	}
	s.publishUpdated(ctx, old, updated)
	result, err := s.withReceivers(ctx, updated)
	if err != nil {
		return Config{}, configError(err, project)
	}
	return result, nil
}

// preconditions are the identity and version an update asserts. Nil fields
// assert nothing.
type preconditions struct {
	ID        *uuid.UUID
	UpdatedAt *time.Time
}

// preconditionsOf reads the UID and resource version of a ProjectConfig, if
// given, as preconditions. A UID that is not one this store issued can match
// no row, so it is a conflict, as it is for Targets.
func preconditionsOf(cfg *kargoapi.ProjectConfig) (preconditions, error) {
	var p preconditions
	if cfg.UID != "" {
		id, err := uuid.Parse(string(cfg.UID))
		if err != nil {
			return p, fmt.Errorf("the ProjectConfig was deleted and recreated: %w", database.ErrConflict)
		}
		p.ID = &id
	}
	if cfg.ResourceVersion != "" {
		updatedAt, err := parseResourceVersion(cfg.ResourceVersion)
		if err != nil {
			return p, err
		}
		p.UpdatedAt = &updatedAt
	}
	return p, nil
}

// updateMissed explains an update that matched no row: either the
// ProjectConfig does not exist, or it no longer satisfies the update's
// preconditions.
func (s *store) updateMissed(
	ctx context.Context,
	project string,
	projectUID string,
	p preconditions,
) error {
	current, err := s.getRow(ctx, s.queries, project, projectUID)
	if err != nil {
		return err
	}
	if err = checkPreconditions(p, current); err != nil {
		return err
	}
	// The row changed again between the update and this read.
	return fmt.Errorf("the ProjectConfig has been modified: %w", database.ErrConflict)
}

// checkPreconditions compares the identity and version a caller asserts, if
// any, against the row about to be written. UpdatedAt is the version: the
// schema advances it with every write to the row.
func checkPreconditions(p preconditions, current database.ProjectConfigRow) error {
	if p.ID != nil && *p.ID != current.ID {
		return fmt.Errorf("the ProjectConfig was deleted and recreated: %w", database.ErrConflict)
	}
	if p.UpdatedAt != nil && !p.UpdatedAt.Equal(current.UpdatedAt) {
		return fmt.Errorf("the ProjectConfig has been modified: %w", database.ErrConflict)
	}
	return nil
}

// Delete removes the Project's ProjectConfig and its receivers. A row created
// for another Project UID is reported as database.ErrNotFound and left alone.
func (s *store) Delete(ctx context.Context, project, projectUID string) error {
	row, err := s.getRow(ctx, s.queries, project, projectUID)
	if err != nil {
		return configError(err, project)
	}
	deleted, err := s.queries.DeleteProjectConfigByID(ctx, row.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = database.ErrNotFound
	}
	if err != nil {
		return configError(err, project)
	}
	s.publishDeleted(ctx, deleted)
	return nil
}

// DeleteForProject removes the ProjectConfig a deleted Project leaves behind:
// the row created for the Project with the given UID. A row created for any
// other UID is left alone. Deleted is published only if a row went away;
// finding none to delete is not an error.
func (s *store) DeleteForProject(
	ctx context.Context,
	project string,
	projectUID string,
) error {
	row, err := s.queries.GetProjectConfig(ctx, project)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return configError(err, project)
	}
	if row.ProjectUID != projectUID {
		return nil
	}
	deleted, err := s.queries.DeleteProjectConfigByID(ctx, row.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return configError(err, project)
	}
	s.publishDeleted(ctx, deleted)
	return nil
}

// AdoptForProject rebinds the Project's ProjectConfig to the Project with
// newUID when the row belongs to an earlier Project of the same name whose
// deletion was not seen, as after a cluster is restored from a backup while
// the database is kept. Only project_uid and updated_at change: the spec,
// generation, refresh token, status and receiver rows stay, so the
// receivers keep their paths and the reconciler, which watches for new
// generations and refresh tokens, is not woken by the Updated event.
//
// It reports whether a row was adopted. No row, or a row already bound to
// newUID, is left alone and nothing is published.
func (s *store) AdoptForProject(
	ctx context.Context,
	project string,
	newUID string,
) (bool, error) {
	var old, adopted database.ProjectConfigRow
	var changed bool
	err := s.inTx(ctx, func(q *database.Queries) error {
		var txErr error
		changed = false
		old, txErr = q.GetProjectConfig(ctx, project)
		if errors.Is(txErr, pgx.ErrNoRows) {
			return nil
		}
		if txErr != nil || old.ProjectUID == newUID {
			return txErr
		}
		adopted, txErr = q.AdoptProjectConfig(ctx, database.AdoptProjectConfigParams{
			ProjectName: project,
			ID:          old.ID,
			ProjectUID:  newUID,
		})
		if errors.Is(txErr, pgx.ErrNoRows) {
			// Replaced, deleted or adopted since it was read.
			return nil
		}
		changed = txErr == nil
		return txErr
	})
	if err != nil {
		return false, configError(err, project)
	}
	if changed {
		s.publishUpdated(ctx, old, adopted)
	}
	return changed, nil
}

// UpdateStatus records the outcome of a reconcile: its conditions, the
// generation and refresh token it observed, and the webhook receivers it
// found, which replace the ProjectConfig's receiver rows. All of it is written
// in one transaction, or none of it.
//
// database.ErrConflict says the row, its generation or its refresh token is
// not the one the reconcile observed, so that a slow reconcile cannot
// overwrite the status of a newer spec, roll the handled refresh back, or
// write a replaced row's status onto its replacement.
//
// A receiver whose path another ProjectConfig owns, or whose name an earlier
// receiver in the status already has, is not stored. The names of those
// receivers are returned, and the rest of the status is written.
func (s *store) UpdateStatus(
	ctx context.Context,
	project string,
	status Status,
) ([]string, error) {
	conditions, err := statusConditions(status.Conditions)
	if err != nil {
		return nil, configError(err, project)
	}
	var observedToken *string
	if status.ObservedRefreshToken != "" {
		observedToken = &status.ObservedRefreshToken
	}
	var old, updated database.ProjectConfigRow
	var skipped []string
	err = s.inTx(ctx, func(q *database.Queries) error {
		var txErr error
		skipped = nil
		if old, txErr = q.GetProjectConfig(ctx, project); txErr != nil {
			if errors.Is(txErr, pgx.ErrNoRows) {
				return database.ErrNotFound
			}
			return txErr
		}
		updated, txErr = q.UpdateProjectConfigStatus(ctx, database.UpdateProjectConfigStatusParams{
			ProjectName:              project,
			ID:                       status.ObservedID,
			StatusConditions:         conditions,
			StatusObservedGeneration: status.ObservedGeneration,
			StatusLastHandledRefresh: observedToken,
			ObservedGeneration:       status.ObservedGeneration,
			ObservedRefreshToken:     observedToken,
		})
		if errors.Is(txErr, pgx.ErrNoRows) {
			return statusMissed(old, status)
		}
		if txErr != nil {
			return txErr
		}
		skipped, txErr = replaceReceivers(ctx, q, updated.ID, status.WebhookReceivers)
		return txErr
	})
	if err != nil {
		return nil, configError(err, project)
	}
	s.publishUpdated(ctx, old, updated)
	return skipped, nil
}

// statusMissed explains a status write that matched no row, from the row as
// read just before it.
func statusMissed(current database.ProjectConfigRow, status Status) error {
	if current.ID != status.ObservedID {
		return fmt.Errorf(
			"the ProjectConfig was deleted and recreated: %w", database.ErrConflict,
		)
	}
	if current.Generation != status.ObservedGeneration {
		return fmt.Errorf(
			"the ProjectConfig is at generation %d, not the observed %d: %w",
			current.Generation, status.ObservedGeneration, database.ErrConflict,
		)
	}
	var currentToken string
	if current.RefreshToken != nil {
		currentToken = *current.RefreshToken
	}
	if currentToken != status.ObservedRefreshToken {
		return fmt.Errorf(
			"a newer refresh than the observed one was requested: %w", database.ErrConflict,
		)
	}
	// The row changed between the read and the write.
	return fmt.Errorf("the ProjectConfig has been modified: %w", database.ErrConflict)
}

// replaceReceivers replaces a ProjectConfig's receiver rows with the given
// receivers and returns the names of those it skipped, as insertReceivers
// does.
func replaceReceivers(
	ctx context.Context,
	q *database.Queries,
	id uuid.UUID,
	receivers []kargoapi.WebhookReceiverDetails,
) ([]string, error) {
	if err := q.DeleteProjectConfigWebhookReceivers(ctx, id); err != nil {
		return nil, err
	}
	return insertReceivers(ctx, q, id, receivers)
}

// insertReceivers stores receiver rows for a ProjectConfig and returns the
// names of those it skipped: those whose paths another ProjectConfig owns,
// and those whose names an earlier receiver already has. A skip does not
// abort the transaction, so the rest of the write goes ahead.
func insertReceivers(
	ctx context.Context,
	q *database.Queries,
	id uuid.UUID,
	receivers []kargoapi.WebhookReceiverDetails,
) ([]string, error) {
	var skipped []string
	for _, receiver := range receivers {
		_, err := q.InsertProjectConfigWebhookReceiver(ctx, database.InsertProjectConfigWebhookReceiverParams{
			ProjectConfigID: id,
			Name:            receiver.Name,
			Path:            receiver.Path,
			URL:             receiver.URL,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			skipped = append(skipped, receiver.Name)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error storing webhook receiver %q: %w", receiver.Name, err)
		}
	}
	return skipped, nil
}

// RequestRefresh records a new refresh token on the Project's ProjectConfig
// and returns it. Neither the spec nor the generation changes; the reconciler,
// told by the Updated event or its next resync, reports the token as
// status.lastHandledRefresh once it has handled it. A row created for another
// Project UID is reported as database.ErrNotFound.
func (s *store) RequestRefresh(ctx context.Context, project, projectUID string) (string, error) {
	// The format is the one the refresh annotation has always carried.
	token := time.Now().UTC().Format(time.RFC3339Nano)
	var old, updated database.ProjectConfigRow
	err := s.inTx(ctx, func(q *database.Queries) error {
		var txErr error
		if old, txErr = s.getRow(ctx, q, project, projectUID); txErr != nil {
			return txErr
		}
		updated, txErr = q.SetProjectConfigRefreshToken(ctx, database.SetProjectConfigRefreshTokenParams{
			ProjectName:  project,
			RefreshToken: token,
		})
		if errors.Is(txErr, pgx.ErrNoRows) || (txErr == nil && updated.ID != old.ID) {
			// The row was deleted, or replaced, since it was read.
			return database.ErrNotFound
		}
		return txErr
	})
	if err != nil {
		return "", configError(err, project)
	}
	s.publishUpdated(ctx, old, updated)
	return token, nil
}

// GetByWebhookReceiverPath returns the ProjectConfig that owns a webhook
// receiver path, or database.ErrNotFound. A ProjectConfig whose Project the
// mirror records under another UID is a leftover and owns nothing; one whose
// Project is not mirrored yet is found.
func (s *store) GetByWebhookReceiverPath(ctx context.Context, path string) (Config, error) {
	row, err := s.queries.GetProjectConfigByWebhookReceiverPath(ctx, path)
	if errors.Is(err, pgx.ErrNoRows) {
		err = database.ErrNotFound
	}
	if err != nil {
		return Config{}, fmt.Errorf("ProjectConfig with webhook receiver path %q: %w", path, err)
	}
	cfg, err := s.withReceivers(ctx, row)
	if err != nil {
		return Config{}, configError(err, row.ProjectName)
	}
	return cfg, nil
}

// Import stores a ProjectConfig read from its Kubernetes resource for the
// Project with the given UID, unless the Project already has one, which is
// never overwritten. It reports whether it stored one. The resource's
// conditions and webhook receivers are copied, so that its receivers keep
// resolving and its state keeps showing until it is next reconciled; the
// observed generation is not, so that it is. A receiver whose path another
// ProjectConfig owns, or whose name an earlier receiver has, is not stored.
func (s *store) Import(
	ctx context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (bool, error) {
	project, err := projectOf(cfg, projectUID)
	if err != nil {
		return false, configError(err, project)
	}
	labels, annotations, spec, err := specColumns(cfg)
	if err != nil {
		return false, configError(err, project)
	}
	conditions, err := statusConditions(cfg.Status.Conditions)
	if err != nil {
		return false, configError(err, project)
	}
	var created database.ProjectConfigRow
	var inserted bool
	err = s.inTx(ctx, func(q *database.Queries) error {
		var txErr error
		inserted = false
		created, txErr = q.ImportProjectConfig(ctx, database.ImportProjectConfigParams{
			ProjectName:      project,
			ProjectUID:       projectUID,
			Labels:           labels,
			Annotations:      annotations,
			Spec:             spec,
			StatusConditions: conditions,
		})
		if errors.Is(txErr, pgx.ErrNoRows) {
			return nil
		}
		if txErr != nil {
			return txErr
		}
		inserted = true
		_, txErr = insertReceivers(ctx, q, created.ID, cfg.Status.WebhookReceivers)
		return txErr
	})
	if err != nil {
		return false, configError(err, project)
	}
	if inserted {
		s.publishCreated(ctx, created)
	}
	return inserted, nil
}

// getRow reads the Project's row through q. A row created for another
// Project UID is a leftover and is reported, as a missing one is, as
// database.ErrNotFound.
func (s *store) getRow(
	ctx context.Context,
	q *database.Queries,
	project string,
	projectUID string,
) (database.ProjectConfigRow, error) {
	row, err := q.GetProjectConfig(ctx, project)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.ProjectUID != projectUID) {
		return database.ProjectConfigRow{}, database.ErrNotFound
	}
	return row, err
}

// withReceivers pairs a row with its receiver rows.
func (s *store) withReceivers(ctx context.Context, row database.ProjectConfigRow) (Config, error) {
	receivers, err := s.queries.ListProjectConfigWebhookReceivers(ctx, row.ID)
	if err != nil {
		return Config{}, fmt.Errorf("error listing webhook receivers: %w", err)
	}
	return Config{ProjectConfigRow: row, Receivers: receivers}, nil
}

// inTx runs fn with queries bound to a new transaction, which it commits if
// fn succeeds and rolls back otherwise.
func (s *store) inTx(ctx context.Context, fn func(*database.Queries) error) error {
	return pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(s.queries.WithTx(tx))
	})
}

// projectOf returns the Project a ProjectConfig belongs to: its name, which
// its namespace, if given, must match. The Project's UID is required.
func projectOf(cfg *kargoapi.ProjectConfig, projectUID string) (string, error) {
	project := cfg.Name
	switch {
	case project == "":
		return "", fmt.Errorf("a ProjectConfig requires a name: %w", database.ErrInvalid)
	case cfg.Namespace != "" && cfg.Namespace != project:
		return project, fmt.Errorf(
			"a ProjectConfig's name must be its Project's, %q: %w", cfg.Namespace, database.ErrInvalid,
		)
	case projectUID == "":
		return project, fmt.Errorf("a ProjectConfig requires its Project's UID: %w", database.ErrInvalid)
	}
	return project, nil
}

// configError names the ProjectConfig an error concerns and translates what
// the database reports into the sentinel errors of pkg/database, so that
// callers can tell a duplicate or a bad value apart from a failure.
func configError(err error, project string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == nameConstraint:
			err = database.ErrAlreadyExists
		case pgErr.Code == "23505" && pgErr.ConstraintName == pathConstraint:
			err = fmt.Errorf("a webhook receiver path is taken: %w", database.ErrConflict)
		case pgErr.Code == "23514" || strings.HasPrefix(pgErr.Code, "22"):
			err = fmt.Errorf("%s: %w", pgErr.Message, database.ErrInvalid)
		}
	}
	// nolint:staticcheck
	return fmt.Errorf("ProjectConfig of Project %q: %w", project, err)
}
