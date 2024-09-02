package garbage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// cleanProjectPromotions steps through all Stages in the specified Project and,
// for each, deletes all Promotions meeting the following criteria:
//   - More than some configurable number of generations older than the oldest
//     Promotion (from the same Stage) in a non-terminal phase.
//   - Older than some configurable minimum age.
func (c *collector) cleanProjectPromotions(ctx context.Context, project string) error {
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
		if err := c.cleanStagePromotionsFn(ctx, project, stage.Name); err != nil {
			stageLogger.Error(err, "error cleaning Promotions to Stage")
			cleanErrCount++
			continue
		}
		stageLogger.Debug("cleaned Promotions to Stage")
	}

	if cleanErrCount > 0 {
		return fmt.Errorf(
			"error cleaning Promotions to one or more Stages in Project %q",
			project,
		)
	}

	return nil
}

func (c *collector) cleanStagePromotions(
	ctx context.Context,
	project string,
	stage string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project,
		"stage", stage,
	)

	promos := kargoapi.PromotionList{}
	if err := c.listPromotionsFn(
		ctx,
		&promos,
		client.InNamespace(project),
		client.MatchingFields{
			kubeclient.PromotionsByStageIndexField: stage,
		},
	); err != nil {
		return fmt.Errorf(
			"error listing Promotions to Stage %q in Project %q: %w",
			stage,
			project,
			err,
		)
	}

	if len(promos.Items) <= c.cfg.MaxRetainedPromotions {
		return nil // Done
	}

	// Sort by creation time desc descending
	slices.SortFunc(promos.Items, func(lhs, rhs kargoapi.Promotion) int {
		return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
	})

	// Step through all Promotions to find the oldest that is not terminal
	oldestNonTerminalIndex := -1
	for i, promo := range promos.Items {
		if !promo.Status.Phase.IsTerminal() {
			oldestNonTerminalIndex = i
		}
	}

	firstToDeleteIndex := oldestNonTerminalIndex + c.cfg.MaxRetainedPromotions + 1
	if firstToDeleteIndex >= len(promos.Items) {
		return nil // Done
	}

	var deleteErrCount int
	for i := firstToDeleteIndex; i < len(promos.Items); i++ {
		promo := promos.Items[i]
		if time.Since(promo.CreationTimestamp.Time) < c.cfg.MinPromotionDeletionAge {
			continue // Not old enough
		}
		promoLogger := logger.WithValues("promotion", promo.Name)
		if err := c.deletePromotionFn(ctx, &promo); err != nil {
			promoLogger.Error(err, "error deleting Promotion")
			deleteErrCount++
		} else {
			promoLogger.Debug("deleted Promotion")
		}
	}

	if deleteErrCount > 0 {
		return fmt.Errorf(
			"error deleting one or more Promotions from Stage %q in Project %q",
			stage,
			project,
		)
	}

	return nil
}
