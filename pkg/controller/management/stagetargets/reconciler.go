// Package stagetargets keeps the stage_targets table, which records which
// Stage governs which Target, in step with the Stages' mirrored selectors and
// the Targets' labels.
package stagetargets

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler"
)

const (
	// reconcileTimeout bounds one Project's reconcile, so a blocked query
	// fails the Project for retry with backoff instead of holding a worker.
	reconcileTimeout = 30 * time.Second
	// batchSize bounds how many pairs one statement inserts or deletes.
	batchSize = 10_000
)

type store interface {
	ListStageSelectors(context.Context, string) ([]database.ListStageSelectorsRow, error)
	ListTargetLabels(context.Context, string) ([]database.ListTargetLabelsRow, error)
	ListStageTargets(context.Context, string) ([]database.StageTargetRow, error)
	InsertStageTargets(context.Context, database.InsertStageTargetsParams) error
	DeleteStageTargets(context.Context, database.DeleteStageTargetsParams) error
}

type stageTargetsReconciler struct {
	store store
}

// reconcile settles one Project's governance: it reads the Project's Stages'
// selectors, its Targets' labels and the rows that stand, works out which
// Stage governs which Target, and writes only the difference. It is handed
// the Project's name, whichever source pushed it, and reads everything
// fresh, so a change that lands while it runs is settled by the next
// reconcile the change feed causes.
func (r *stageTargetsReconciler) reconcile(
	ctx context.Context,
	project string,
) (reconciler.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()
	logger := logging.LoggerFromContext(ctx).WithValues("project", project)
	ctx = logging.ContextWithLogger(ctx, logger)

	stages, err := r.store.ListStageSelectors(ctx, project)
	if err != nil {
		return reconciler.Result{}, fmt.Errorf("error listing stage selectors: %w", err)
	}
	targets, err := r.store.ListTargetLabels(ctx, project)
	if err != nil {
		return reconciler.Result{}, fmt.Errorf("error listing target labels: %w", err)
	}
	rows, err := r.store.ListStageTargets(ctx, project)
	if err != nil {
		return reconciler.Result{}, fmt.Errorf("error listing stage targets: %w", err)
	}

	desired := governance(ctx, stages, targets)
	var stale []pair
	for _, row := range rows {
		current := pair{stageID: row.StageID, targetID: row.TargetID}
		if _, keep := desired[current]; keep {
			delete(desired, current)
			continue
		}
		stale = append(stale, current)
	}
	added := make([]pair, 0, len(desired))
	for missing := range desired {
		added = append(added, missing)
	}

	for batch := range batches(stale) {
		stageIDs, targetIDs := columns(batch)
		if err = r.store.DeleteStageTargets(ctx, database.DeleteStageTargetsParams{
			StageIds:  stageIDs,
			TargetIds: targetIDs,
		}); err != nil {
			return reconciler.Result{}, fmt.Errorf("error deleting stage targets: %w", err)
		}
	}
	for batch := range batches(added) {
		stageIDs, targetIDs := columns(batch)
		if err = r.store.InsertStageTargets(ctx, database.InsertStageTargetsParams{
			StageIds:  stageIDs,
			TargetIds: targetIDs,
		}); err != nil {
			return reconciler.Result{}, fmt.Errorf("error inserting stage targets: %w", err)
		}
	}
	if len(stale) > 0 || len(added) > 0 {
		logger.Debug("settled stage targets", "added", len(added), "removed", len(stale))
	}
	return reconciler.Result{}, nil
}

// batches yields the pairs in runs of at most batchSize.
func batches(pairs []pair) func(func([]pair) bool) {
	return func(yield func([]pair) bool) {
		for start := 0; start < len(pairs); start += batchSize {
			if !yield(pairs[start:min(start+batchSize, len(pairs))]) {
				return
			}
		}
	}
}

// columns splits pairs into the two parallel arrays the queries take.
func columns(pairs []pair) ([]string, []uuid.UUID) {
	stageIDs := make([]string, len(pairs))
	targetIDs := make([]uuid.UUID, len(pairs))
	for i, p := range pairs {
		stageIDs[i], targetIDs[i] = p.stageID, p.targetID
	}
	return stageIDs, targetIDs
}
