package garbage

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// TODO: This needs to be fixed per https://github.com/akuity/kargo/issues/1588
func (c *collector) cleanProjectPromotions(ctx context.Context, project string) error {
	logger := logging.LoggerFromContext(ctx).WithField("project", project)

	promos := kargoapi.PromotionList{}
	if err := c.listPromotionsFn(
		ctx,
		&promos,
		client.InNamespace(project),
	); err != nil {
		return errors.Wrapf(err, "error listing Promotions for Project %q", project)
	}

	if len(promos.Items) <= c.cfg.MaxRetainedPromotions {
		return nil // Done
	}

	// Sort Promotions by creation time
	sort.Sort(promosByCreation(promos.Items))

	// Delete oldest Promotions (in terminal phases only) that are in excess of
	// MaxRetainedPromotions
	var deleteErrCount int
	for i := c.cfg.MaxRetainedPromotions; i < len(promos.Items); i++ {
		promo := promos.Items[i]
		if promo.Status.Phase.IsTerminal() {
			promoLogger := logger.WithField("promotion", promo.Name)
			if err := c.deletePromotionFn(ctx, &promo); err != nil {
				promoLogger.Errorf("error deleting Promotion: %s", err)
				deleteErrCount++
			} else {
				promoLogger.Debug("deleted Promotion")
			}
		}
	}

	if deleteErrCount > 0 {
		return errors.Errorf(
			"error deleting one or more Promotions from Project %q",
			project,
		)
	}

	return nil
}
