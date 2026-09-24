package database

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// PromotionRequestTargetRow is one Target of a PromotionRequest: its
// membership in the snapshot taken when the request was created, and its
// outcome so far.
type PromotionRequestTargetRow = ListPromotionRequestTargetsRow

// PromotionRequestSnapshot is a PromotionRequest row together with the names
// of the resources it references and its Target rows, all read from the same
// database snapshot.
//
// A snapshot is also the body of every event about a PromotionRequest, so its
// JSON form is part of the contract between the components that publish and
// consume them. The row's columns appear at the top level.
type PromotionRequestSnapshot struct {
	PromotionRequest
	ProjectName string                      `json:"project_name"`
	Stage       string                      `json:"stage"`
	Freight     string                      `json:"freight"`
	Targets     []PromotionRequestTargetRow `json:"targets"`
}

// PromotionRequestCreate describes a PromotionRequest to insert. Targets holds
// the names of the Targets the request fans out to, in order. Every name must
// belong to a Target in the Project.
type PromotionRequestCreate struct {
	CreatePromotionRequestParams
	Targets []string
}

// promotionRequestRow is the shape shared by every query that reads
// PromotionRequests with the names of the resources they reference.
type promotionRequestRow struct {
	PromotionRequest PromotionRequest
	ProjectName      string
	Stage            string
	Freight          string
}

func (s *store) CreatePromotionRequest(
	ctx context.Context,
	create PromotionRequestCreate,
) (PromotionRequestSnapshot, error) {
	var snapshot PromotionRequestSnapshot
	err := s.transact(ctx, func(txCtx context.Context, q *Queries) error {
		row, err := q.CreatePromotionRequest(txCtx, create.CreatePromotionRequestParams)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf(
				"Project %q, Stage %q or Freight %q: %w",
				create.ProjectName, create.Stage, create.Freight, ErrNotMirrored,
			)
		}
		if err != nil {
			return fmt.Errorf("error inserting promotion request: %w", err)
		}
		targets, err := q.ListTargetsByName(txCtx, ListTargetsByNameParams{
			ProjectName: create.ProjectName,
			Names:       create.Targets,
		})
		if err != nil {
			return fmt.Errorf("error resolving Targets: %w", err)
		}
		byName := make(map[string]Target, len(targets))
		for _, target := range targets {
			byName[target.Name] = target
		}
		snapshot = PromotionRequestSnapshot{
			PromotionRequest: row,
			ProjectName:      create.ProjectName,
			Stage:            create.Stage,
			Freight:          create.Freight,
			Targets:          make([]PromotionRequestTargetRow, 0, len(create.Targets)),
		}
		for i, name := range create.Targets {
			target, ok := byName[name]
			if !ok {
				return fmt.Errorf("Target %q in Project %q: %w", name, create.ProjectName, ErrNotMirrored)
			}
			targetRow := PromotionRequestTargetRow{
				PromotionRequestID: row.ID,
				TargetID:           target.ID,
				Name:               name,
				Ordinal:            int64(i),
			}
			if err = q.InsertPromotionRequestTarget(txCtx, InsertPromotionRequestTargetParams{
				PromotionRequestID: targetRow.PromotionRequestID,
				TargetID:           targetRow.TargetID,
				Ordinal:            targetRow.Ordinal,
			}); err != nil {
				return fmt.Errorf("error inserting promotion request target %q: %w", name, err)
			}
			snapshot.Targets = append(snapshot.Targets, targetRow)
		}
		return nil
	})
	return snapshot, err
}

func (s *store) GetPromotionRequest(
	ctx context.Context,
	params GetPromotionRequestParams,
) (PromotionRequestSnapshot, error) {
	var snapshots []PromotionRequestSnapshot
	err := s.readSnapshot(ctx, func(txCtx context.Context, q *Queries) error {
		row, err := q.GetPromotionRequest(txCtx, params)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf(
				"PromotionRequest %q in Project %q: %w", params.Name, params.ProjectName, ErrNotFound,
			)
		}
		if err != nil {
			return err
		}
		snapshots, err = collectSnapshots(txCtx, q, []promotionRequestRow{promotionRequestRow(row)})
		return err
	})
	if err != nil {
		return PromotionRequestSnapshot{}, err
	}
	return snapshots[0], nil
}

func (s *store) ListPromotionRequests(
	ctx context.Context,
	projectName string,
) ([]PromotionRequestSnapshot, error) {
	var snapshots []PromotionRequestSnapshot
	err := s.readSnapshot(ctx, func(txCtx context.Context, q *Queries) error {
		rows, err := q.ListPromotionRequests(txCtx, projectName)
		if err != nil {
			return err
		}
		shared := make([]promotionRequestRow, len(rows))
		for i, row := range rows {
			shared[i] = promotionRequestRow(row)
		}
		snapshots, err = collectSnapshots(txCtx, q, shared)
		return err
	})
	return snapshots, err
}

func (s *store) ListPromotionRequestsByStage(
	ctx context.Context,
	params ListPromotionRequestsByStageParams,
) ([]PromotionRequestSnapshot, error) {
	var snapshots []PromotionRequestSnapshot
	err := s.readSnapshot(ctx, func(txCtx context.Context, q *Queries) error {
		rows, err := q.ListPromotionRequestsByStage(txCtx, params)
		if err != nil {
			return err
		}
		shared := make([]promotionRequestRow, len(rows))
		for i, row := range rows {
			shared[i] = promotionRequestRow(row)
		}
		snapshots, err = collectSnapshots(txCtx, q, shared)
		return err
	})
	return snapshots, err
}

func (s *store) GetPromotionRequestByID(
	ctx context.Context,
	id uuid.UUID,
) (PromotionRequestSnapshot, error) {
	var snapshots []PromotionRequestSnapshot
	err := s.readSnapshot(ctx, func(txCtx context.Context, q *Queries) error {
		row, err := q.GetPromotionRequestByID(txCtx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("PromotionRequest %s: %w", id, ErrNotFound)
		}
		if err != nil {
			return err
		}
		snapshots, err = collectSnapshots(txCtx, q, []promotionRequestRow{promotionRequestRow(row)})
		return err
	})
	if err != nil {
		return PromotionRequestSnapshot{}, err
	}
	return snapshots[0], nil
}

// ListOpenPromotionRequestIDs returns up to limit ids of Pending and Running
// PromotionRequests that sort after the given one, in id order. It pages the
// PromotionRequest reconciler's resync, and satisfies dbreconcile.Lister.
func (s *store) ListOpenPromotionRequestIDs(
	ctx context.Context,
	after uuid.UUID,
	limit int,
) ([]uuid.UUID, error) {
	if limit < 1 || limit > math.MaxInt32 {
		return nil, fmt.Errorf("invalid limit %d", limit)
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.ListOpenPromotionRequestIDs(ctx, ListOpenPromotionRequestIDsParams{
		AfterID:  after,
		RowLimit: int32(limit),
	})
}

func (s *store) PromotionRequestExists(
	ctx context.Context,
	params PromotionRequestExistsParams,
) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return s.queries.PromotionRequestExists(ctx, params)
}

// UpdatePromotionRequestStatus writes the request's phase, message and
// timestamps, and the outcome of every Target named in the status. Targets the
// status does not name are left as they are; names that are not Targets of the
// request are ignored.
//
// It returns the request as it was immediately before the write and as the
// write left it. The row is locked for the duration, so the two describe
// exactly this write even when others race it; they are what an event about
// the write should carry.
func (s *store) UpdatePromotionRequestStatus(
	ctx context.Context,
	id uuid.UUID,
	status kargoapi.PromotionRequestStatus,
) (PromotionRequestSnapshot, PromotionRequestSnapshot, error) {
	var before, after PromotionRequestSnapshot
	err := s.transact(ctx, func(txCtx context.Context, q *Queries) error {
		row, err := q.GetPromotionRequestByIDForUpdate(txCtx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("PromotionRequest %s: %w", id, ErrNotFound)
		}
		if err != nil {
			return fmt.Errorf("error locking promotion request: %w", err)
		}
		locked := promotionRequestRow(row)
		snapshots, err := collectSnapshots(txCtx, q, []promotionRequestRow{locked})
		if err != nil {
			return err
		}
		before = snapshots[0]

		updated, err := q.UpdatePromotionRequestStatus(txCtx, UpdatePromotionRequestStatusParams{
			ID:         id,
			Phase:      string(status.Phase),
			Message:    status.Message,
			StartedAt:  timestamptz(status.StartedAt),
			FinishedAt: timestamptz(status.FinishedAt),
		})
		if err != nil {
			return fmt.Errorf("error updating promotion request status: %w", err)
		}
		if updated == 0 {
			return fmt.Errorf("PromotionRequest %s: %w", id, ErrNotFound)
		}
		byName := make(map[string]PromotionRequestTargetRow, len(before.Targets))
		for _, target := range before.Targets {
			byName[target.Name] = target
		}
		for _, targetStatus := range status.Targets {
			target, ok := byName[targetStatus.Name]
			if !ok {
				continue
			}
			if err = q.UpdatePromotionRequestTarget(txCtx, UpdatePromotionRequestTargetParams{
				PromotionRequestID: id,
				TargetID:           target.TargetID,
				Promotion:          targetStatus.Promotion,
				Phase:              string(targetStatus.Phase),
			}); err != nil {
				return fmt.Errorf("error updating promotion request target %q: %w", targetStatus.Name, err)
			}
		}

		// Read back through the same query, so the two versions differ only
		// in what the write changed.
		if row, err = q.GetPromotionRequestByIDForUpdate(txCtx, id); err != nil {
			return fmt.Errorf("error reading updated promotion request: %w", err)
		}
		if snapshots, err = collectSnapshots(txCtx, q, []promotionRequestRow{promotionRequestRow(row)}); err != nil {
			return err
		}
		after = snapshots[0]
		return nil
	})
	if err != nil {
		return PromotionRequestSnapshot{}, PromotionRequestSnapshot{}, err
	}
	return before, after, nil
}

// readSnapshot runs read in a read-only transaction at repeatable read
// isolation, so that a request and its Target rows come from one snapshot.
func (s *store) readSnapshot(ctx context.Context, read func(context.Context, *Queries) error) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	}, func(tx pgx.Tx) error {
		return read(ctx, s.queries.WithTx(tx))
	})
}

// collectSnapshots attaches each request's Target rows to it.
func collectSnapshots(
	ctx context.Context,
	q *Queries,
	rows []promotionRequestRow,
) ([]PromotionRequestSnapshot, error) {
	snapshots := make([]PromotionRequestSnapshot, len(rows))
	ids := make([]uuid.UUID, len(rows))
	byID := make(map[uuid.UUID]*PromotionRequestSnapshot, len(rows))
	for i, row := range rows {
		snapshots[i] = PromotionRequestSnapshot{
			PromotionRequest: row.PromotionRequest,
			ProjectName:      row.ProjectName,
			Stage:            row.Stage,
			Freight:          row.Freight,
		}
		ids[i] = row.PromotionRequest.ID
		byID[ids[i]] = &snapshots[i]
	}
	if len(ids) == 0 {
		return snapshots, nil
	}
	targets, err := q.ListPromotionRequestTargets(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("error listing promotion request targets: %w", err)
	}
	for _, target := range targets {
		snapshot := byID[target.PromotionRequestID]
		snapshot.Targets = append(snapshot.Targets, target)
	}
	return snapshots, nil
}

// PromotionRequestFromSnapshot presents a snapshot in the shape of the
// PromotionRequest resource, so that code written against the resource can
// consume rows unchanged. The row's id becomes the UID and its last update
// becomes the resource version. The request's Targets and their outcomes are
// derived from its Target rows: a Target appears in the status once something
// has acted on it, and the summary counts those outcomes.
func PromotionRequestFromSnapshot(snapshot PromotionRequestSnapshot) kargoapi.PromotionRequest {
	var annotations map[string]string
	if snapshot.CreatedBy != "" {
		annotations = map[string]string{kargoapi.AnnotationKeyCreateActor: snapshot.CreatedBy}
	}
	// Never nil: spec.targets is a required field. An empty list is meaningful;
	// it records that the Stage governed no Targets when the request was made.
	specTargets := make([]kargoapi.PromotionRequestTarget, len(snapshot.Targets))
	var statusTargets []kargoapi.PromotionRequestTargetStatus
	var summary *kargoapi.PromotionRequestSummary
	for i, target := range snapshot.Targets {
		specTargets[i] = kargoapi.PromotionRequestTarget{Name: target.Name}
		if target.Phase == "" && target.Promotion == "" {
			continue
		}
		phase := kargoapi.PromotionPhase(target.Phase)
		statusTargets = append(statusTargets, kargoapi.PromotionRequestTargetStatus{
			Name:      target.Name,
			Promotion: target.Promotion,
			Phase:     phase,
		})
		if summary == nil {
			summary = &kargoapi.PromotionRequestSummary{}
		}
		switch phase {
		case kargoapi.PromotionPhasePending:
			summary.Pending++
		case kargoapi.PromotionPhaseRunning:
			summary.Running++
		case kargoapi.PromotionPhaseSucceeded:
			summary.Succeeded++
		case kargoapi.PromotionPhaseFailed:
			summary.Failed++
		case kargoapi.PromotionPhaseErrored:
			summary.Errored++
		case kargoapi.PromotionPhaseAborted:
			summary.Aborted++
		}
	}
	return kargoapi.PromotionRequest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "PromotionRequest",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         snapshot.ProjectName,
			Name:              snapshot.Name,
			UID:               types.UID(snapshot.ID.String()),
			ResourceVersion:   resourceVersion(snapshot.UpdatedAt),
			CreationTimestamp: metav1.NewTime(snapshot.CreatedAt),
			Labels:            map[string]string{kargoapi.LabelKeyStage: snapshot.Stage},
			Annotations:       annotations,
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   snapshot.Stage,
			Freight: snapshot.Freight,
			Targets: specTargets,
		},
		Status: kargoapi.PromotionRequestStatus{
			Phase:      kargoapi.PromotionRequestPhase(snapshot.Phase),
			Message:    snapshot.Message,
			Targets:    statusTargets,
			Summary:    summary,
			StartedAt:  optionalTime(snapshot.StartedAt),
			FinishedAt: optionalTime(snapshot.FinishedAt),
		},
	}
}

// PromotionRequestsFromSnapshots converts every snapshot with
// PromotionRequestFromSnapshot.
func PromotionRequestsFromSnapshots(snapshots []PromotionRequestSnapshot) []kargoapi.PromotionRequest {
	requests := make([]kargoapi.PromotionRequest, len(snapshots))
	for i, snapshot := range snapshots {
		requests[i] = PromotionRequestFromSnapshot(snapshot)
	}
	return requests
}

// NewPromotionRequestCreate is the inverse of PromotionRequestFromSnapshot for
// a request that has not been stored yet: it describes the row to insert for
// the given resource.
func NewPromotionRequestCreate(request *kargoapi.PromotionRequest) PromotionRequestCreate {
	targets := make([]string, len(request.Spec.Targets))
	for i, target := range request.Spec.Targets {
		targets[i] = target.Name
	}
	return PromotionRequestCreate{
		CreatePromotionRequestParams: CreatePromotionRequestParams{
			ProjectName: request.Namespace,
			Name:        request.Name,
			Stage:       request.Spec.Stage,
			Freight:     request.Spec.Freight,
			CreatedBy:   request.Annotations[kargoapi.AnnotationKeyCreateActor],
		},
		Targets: targets,
	}
}

func optionalTime(ts pgtype.Timestamptz) *metav1.Time {
	if !ts.Valid {
		return nil
	}
	t := metav1.NewTime(ts.Time)
	return &t
}

func timestamptz(t *metav1.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t.Time, Valid: true}
}
