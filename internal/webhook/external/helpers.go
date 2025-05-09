package external

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

type refreshResult struct {
	totalWarehouses int
	numFailures     int
}

func refresh(
	ctx context.Context,
	c client.Client,
	repoName string,
) (*refreshResult, error) {
	logger := logging.LoggerFromContext(ctx)
	var warehouses v1alpha1.WarehouseList
	err := c.List(
		ctx,
		&warehouses,
		client.MatchingFields{
			indexer.WarehousesBySubscribedURLsField: repoName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w", err)
	}

	logger.Debug("listed warehouses",
		"num-warehouses", len(warehouses.Items),
	)

	var total, numRefreshFailures int
	for _, wh := range warehouses.Items {
		total++
		_, err = api.RefreshWarehouse(
			ctx,
			c,
			types.NamespacedName{
				Namespace: wh.GetNamespace(),
				Name:      wh.GetName(),
			},
		)
		if err != nil {
			logger.Error(err, "failed to refresh warehouse",
				"warehouse", wh.GetName(),
				"error", err.Error(),
			)
			numRefreshFailures++
		} else {
			logger.Debug("successfully patched annotations",
				"warehouse", wh.GetName(),
			)
		}
	}
	return &refreshResult{
		numFailures:     numRefreshFailures,
		totalWarehouses: len(warehouses.Items),
	}, nil
}
