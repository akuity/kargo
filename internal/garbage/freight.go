package garbage

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// cleanProjectFreight steps through all Warehouses in the specified Project
// and, for each, deletes all Freight meeting the following criteria:
//   - More than some configurable number of generations older than the oldest
//     Freight (from the same Warehouse) that remains in use.
//   - Older than some configurable minimum age.
func (c *collector) cleanProjectFreight(ctx context.Context, project string) error {
	logger := logging.LoggerFromContext(ctx).WithField("project", project)

	warehouses := &kargoapi.WarehouseList{}
	if err := c.listWarehousesFn(
		ctx,
		warehouses,
		client.InNamespace(project),
	); err != nil {
		return errors.Wrapf(err, "error listing Warehouses in Project %q", project)
	}

	var cleanErrCount int
	for _, warehouse := range warehouses.Items {
		warehouseLogger := logger.WithField("warehouse", warehouse.Name)
		if err := c.cleanWarehouseFreightFn(ctx, project, warehouse.Name); err != nil {
			warehouseLogger.Error("error cleaning Freight from Warehouse")
			cleanErrCount++
			continue
		}
		warehouseLogger.Debug("cleaned Freight from Warehouse")
	}

	if cleanErrCount > 0 {
		return errors.Errorf(
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
	logger := logging.LoggerFromContext(ctx).WithFields(logrus.Fields{
		"project":   project,
		"warehouse": warehouse,
	})

	freight := kargoapi.FreightList{}
	if err := c.listFreightFn(
		ctx,
		&freight,
		client.InNamespace(project),
		client.MatchingFields{
			kubeclient.FreightByWarehouseIndexField: warehouse,
		},
	); err != nil {
		return errors.Wrapf(
			err,
			"error listing Freight from Warehouse %q in Project %q",
			warehouse,
			project,
		)
	}

	if len(freight.Items) <= c.cfg.MaxRetainedFreight {
		return nil // Done
	}

	// Sort Freight by creation time
	sort.Sort(freightByCreation(freight.Items))

	// Step through all Freight and find the oldest that is still in use
	oldestInUseIndex := -1
	for i, f := range freight.Items {
		stages := kargoapi.StageList{}
		if err := c.listStagesFn(
			ctx,
			&stages,
			client.InNamespace(project),
			client.MatchingFields{
				kubeclient.StagesByFreightIndexField: f.ID,
			},
		); err != nil {
			logger.WithField("freight", f).Error("error listing Stages using Freight")
			return errors.Wrapf(
				err,
				"error listing Stages in Project %q using Freight %q",
				project,
				f.Name,
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
		freightLogger := logger.WithField("freight", f.Name)
		if err := c.deleteFreightFn(ctx, &f); err != nil {
			freightLogger.Errorf("error deleting Freight: %s", err)
			deleteErrCount++
		} else {
			freightLogger.Debug("deleted Freight")
		}
	}

	if deleteErrCount > 0 {
		return errors.Errorf(
			"error deleting one or more Freight from Project %q Warehouse %q",
			project,
			warehouse,
		)
	}

	return nil
}
