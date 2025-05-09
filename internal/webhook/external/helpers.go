package external

import (
	"context"
	"fmt"
	"io"
	"net/http"

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
	l *logging.Logger,
	repoName string,
) (*refreshResult, error) {
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

	l.Debug("listed warehouses",
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
			l.Error(err, "failed to refresh warehouse",
				"warehouse", wh.GetName(),
				"error", err.Error(),
			)
			numRefreshFailures++
		} else {
			l.Debug("successfully patched annotations",
				"warehouse", wh.GetName(),
			)
		}
	}
	return &refreshResult{
		numFailures:     numRefreshFailures,
		totalWarehouses: len(warehouses.Items),
	}, nil
}

func limitRead(r io.Reader) ([]byte, int, error) {
	const maxBytes = 2 << 20
	lr := io.LimitReader(r, maxBytes)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to read request")
	}

	// If we read exactly the maximum, the body might be larger
	if len(bodyBytes) == maxBytes {
		// Try to read one more byte
		buf := make([]byte, 1)
		var n int
		if n, err = r.Read(buf); err != nil && err != io.EOF {
			return nil,
				http.StatusInternalServerError,
				fmt.Errorf("failed to check for additional content: %w", err)
		}
		if n > 0 {
			return nil,
				http.StatusRequestEntityTooLarge,
				fmt.Errorf("response body exceeds maximum size of %d bytes", maxBytes)
		}
	}
	return bodyBytes, http.StatusOK, nil
}
