package external

import (
	"context"
	"crypto/sha256"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
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

func refreshWarehouses(
	ctx context.Context,
	c client.Client,
	namespace string,
	repoURL string,
) (*refreshResult, error) {
	logger := logging.LoggerFromContext(ctx)
	var warehouses v1alpha1.WarehouseList
	err := c.List(
		ctx,
		&warehouses,
		client.InNamespace(namespace),
		client.MatchingFields{
			indexer.WarehousesBySubscribedURLsField: repoURL,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list warehouses: %w", err)
	}

	logger.Debug("listed warehouses",
		"count", len(warehouses.Items),
	)

	var failures int
	for _, wh := range warehouses.Items {
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
			failures++
		} else {
			logger.Debug("successfully patched annotations",
				"warehouse", wh.GetName(),
			)
		}
	}
	return &refreshResult{
		failures:  failures,
		successes: len(warehouses.Items) - failures,
	}, nil
}

func GenerateWebhookPath(project, provider, token string) string {
	input := []byte(project + provider + token)
	h := sha256.New()
	h.Write(input)
	return fmt.Sprintf("/%x", h.Sum(nil))
}
