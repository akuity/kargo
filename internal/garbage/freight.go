package garbage

import (
	"context"
	"fmt"
	"slices"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

// cleanProjectFreight steps through all Warehouses in the specified Project
// and, for each, deletes all Freight meeting the following criteria:
//   - More than some configurable number of generations older than the oldest
//     Freight (from the same Warehouse) that remains in use.
//   - Older than some configurable minimum age.
func (c *collector) cleanProjectFreight(ctx context.Context, project string) error {
	logger := logging.LoggerFromContext(ctx).WithValues("project", project)

	warehouses := &kargoapi.WarehouseList{}
	if err := c.listWarehousesFn(
		ctx,
		warehouses,
		client.InNamespace(project),
	); err != nil {
		return fmt.Errorf("error listing Warehouses in Project %q: %w", project, err)
	}

	var cleanErrCount int
	for _, warehouse := range warehouses.Items {
		warehouseLogger := logger.WithValues("warehouse", warehouse.Name)
		if err := c.cleanWarehouseFreightFn(ctx, project, warehouse.Name); err != nil {
			warehouseLogger.Error(err, "error cleaning Freight from Warehouse")
			cleanErrCount++
			continue
		}
		warehouseLogger.Debug("cleaned Freight from Warehouse")
	}

	if cleanErrCount > 0 {
		return fmt.Errorf(
			"error cleaning Freight from one or more Warehouses in Project %q",
			project,
		)
	}

	return nil
}

// cleanWarehouseFreight deletes all Freight from the specified Project and
// Warehouse that meet the following criteria:
//   - More than some configurable number of generations older than the oldest
//     Freight (from the same Warehouse) that remains in use.
//   - Older than some configurable minimum age.
func (c *collector) cleanWarehouseFreight(
	ctx context.Context,
	project string,
	warehouse string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"project", project,
		"warehouse", warehouse,
	)

	freight := kargoapi.FreightList{}
	if err := c.listFreightFn(
		ctx,
		&freight,
		client.InNamespace(project),
		client.MatchingFields{
			indexer.FreightByWarehouseIndexField: warehouse,
		},
	); err != nil {
		return fmt.Errorf(
			"error listing Freight from Warehouse %q in Project %q: %w",
			warehouse,
			project,
			err,
		)
	}

	if len(freight.Items) <= c.cfg.MaxRetainedFreight {
		return nil // Done
	}

	// Sort by creation timestamp descending
	slices.SortFunc(freight.Items, func(lhs, rhs kargoapi.Freight) int {
		return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
	})

	// Step through all Freight and find the oldest that is still in use
	oldestInUseIndex := -1
	for i, f := range freight.Items {
		stages := kargoapi.StageList{}
		if err := c.listStagesFn(
			ctx,
			&stages,
			client.InNamespace(project),
			client.MatchingFields{
				indexer.StagesByFreightIndexField: f.Name,
			},
		); err != nil {
			logger.Error(
				err, "error listing Stages using Freight",
				"freight", f,
			)
			return fmt.Errorf(
				"error listing Stages in Project %q using Freight %q: %w",
				project,
				f.Name,
				err,
			)
		}
		if len(stages.Items) > 0 {
			oldestInUseIndex = i
		}
	}

	firstToDeleteIndex := oldestInUseIndex + c.cfg.MaxRetainedFreight + 1
	if firstToDeleteIndex >= len(freight.Items) {
		return nil // Done
	}

	var deleteErrCount int
	for i := firstToDeleteIndex; i < len(freight.Items); i++ {
		f := freight.Items[i]
		if time.Since(f.CreationTimestamp.Time) < c.cfg.MinFreightDeletionAge {
			continue // Not old enough
		}
		freightLogger := logger.WithValues("freight", f.Name)
		if err := c.deleteFreightFn(ctx, &f); err != nil {
			freightLogger.Error(err, "error deleting Freight")
			deleteErrCount++
		} else {
			freightLogger.Debug("deleted Freight")
		}
	}

	if deleteErrCount > 0 {
		return fmt.Errorf(
			"error deleting one or more Freight from Warehouse %q in Project %q",
			warehouse,
			project,
		)
	}

	return nil
}
