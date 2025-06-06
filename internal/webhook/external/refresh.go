package external

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

type refreshResult struct {
	successes int
	failures  int
}

// refreshWarehouses refreshes all Warehouses in the given namespace that are
// subscribed to the given repository URL. If the namespace is empty, all
// Warehouses in the cluster subscribed to the given repository URL are
// refreshed. Note: Callers are responsible for normalizing the provided
// repository URL.
func refreshWarehouses(
	ctx context.Context,
	c client.Client,
	project string,
	repoURL string,
) (*refreshResult, error) {
	logger := logging.LoggerFromContext(ctx)

	listOpts := make([]client.ListOption, 1, 2)
	listOpts[0] = client.MatchingFields{
		indexer.WarehousesBySubscribedURLsField: repoURL,
	}
	if project != "" {
		listOpts = append(listOpts, client.InNamespace(project))
	}

	warehouses := v1alpha1.WarehouseList{}
	if err := c.List(ctx, &warehouses, listOpts...); err != nil {
		return nil, fmt.Errorf("error listing Warehouses: %w", err)
	}

	logger.Debug("found Warehouses to refresh", "count", len(warehouses.Items))

	var failures int
	for _, wh := range warehouses.Items {
		objKey := client.ObjectKeyFromObject(&wh)
		if _, err := api.RefreshWarehouse(ctx, c, objKey); err != nil {
			logger.Error(err, "error refreshing Warehouse", "objectKey", objKey)
			failures++
		} else {
			logger.Debug("refreshed Warehouse", "objectKey", objKey)
		}
	}
	return &refreshResult{
		failures:  failures,
		successes: len(warehouses.Items) - failures,
	}, nil
}
