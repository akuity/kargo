package garbage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

// cleanProjectPromotionRequests steps through all Stages in the specified
// Project and, for each, deletes all PromotionRequests meeting the following
// criteria:
//   - More than some configurable number of generations older than the oldest
//     PromotionRequest (from the same Stage) in a non-terminal phase.
//   - Older than some configurable minimum age.
func (c *collector) cleanProjectPromotionRequests(
	ctx context.Context,
	project string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues("project", project)

	stages := &kargoapi.StageList{}
	if err := c.listStagesFn(
		ctx,
		stages,
		client.InNamespace(project),
	); err != nil {
		return fmt.Errorf("error listing Stages in Project %q: %w", project, err)
	}

	var cleanErrCount int
	for _, stage := range stages.Items {
		stageLogger := logger.WithValues("stage", stage.Name)
		if err := c.cleanStagePromotionRequestsFn(ctx, project, stage.Name); err != nil {
			stageLogger.Error(err, "error cleaning PromotionRequests for Stage")
			cleanErrCount++
			continue
		}
		stageLogger.Debug("cleaned PromotionRequests for Stage")
	}

	if cleanErrCount > 0 {
		return fmt.Errorf(
			"error cleaning PromotionRequests for one or more Stages in Project %q",
			project,
		)
	}

	return nil
}

func (c *collector) cleanStagePromotionRequests(
	ctx context.Context,
	project string,
	stage string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project,
		"stage", stage,
	)

	requests := kargoapi.PromotionRequestList{}
	if err := c.listPromotionRequestsFn(
		ctx,
		&requests,
		client.InNamespace(project),
		client.MatchingFields{
			indexer.PromotionRequestsByStageField: stage,
		},
	); err != nil {
		return fmt.Errorf(
			"error listing PromotionRequests for Stage %q in Project %q: %w",
			stage,
			project,
			err,
		)
	}

	if len(requests.Items) <= c.cfg.MaxRetainedPromotionRequests {
		return nil // Done
	}

	// Sort by creation time descending
	slices.SortFunc(
		requests.Items,
		func(lhs, rhs kargoapi.PromotionRequest) int {
			return rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time)
		},
	)

	// Step through all PromotionRequests to find the oldest that is not terminal
	oldestNonTerminalIndex := -1
	for i, request := range requests.Items {
		if !request.Status.Phase.IsTerminal() {
			oldestNonTerminalIndex = i
		}
	}

	firstToDeleteIndex :=
		oldestNonTerminalIndex + c.cfg.MaxRetainedPromotionRequests + 1
	if firstToDeleteIndex >= len(requests.Items) {
		return nil // Done
	}

	var deleteErrCount int
	for i := firstToDeleteIndex; i < len(requests.Items); i++ {
		request := requests.Items[i]
		if time.Since(request.CreationTimestamp.Time) <
			c.cfg.MinPromotionRequestDeletionAge {
			continue // Not old enough
		}
		requestLogger := logger.WithValues("promotionRequest", request.Name)
		if err := c.deletePromotionRequestFn(ctx, &request); err != nil {
			requestLogger.Error(err, "error deleting PromotionRequest")
			deleteErrCount++
		} else {
			requestLogger.Debug("deleted PromotionRequest")
		}
	}

	if deleteErrCount > 0 {
		return fmt.Errorf(
			"error deleting one or more PromotionRequests for Stage %q in Project %q",
			stage,
			project,
		)
	}

	return nil
}
