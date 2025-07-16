package external

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

// refreshWarehouses refreshes all Warehouses in the given namespace that are
// subscribed to any of the given repository URLs. If the namespace is empty,
// all Warehouses in the cluster subscribed to the given repository URLs are
// refreshed. Note: Callers are responsible for normalizing the provided
// repository URLs.
func refreshWarehouses(
	ctx context.Context,
	w http.ResponseWriter,
	c client.Client,
	project string,
	repoURLs ...string,
) {
	logger := logging.LoggerFromContext(ctx)

	// De-dupe repository URLs
	slices.Sort(repoURLs)
	repoURLs = slices.Compact(repoURLs)
	// If there had been any empty strings in the slice, after sorting and
	// compacting, at most the zero element will be empty. If it is, remove it.
	if repoURLs[0] == "" {
		repoURLs = repoURLs[1:]
	}

	warehouses := []kargoapi.Warehouse{}

	for _, repoURL := range repoURLs {
		repoLogger := logger.WithValues("repositoryURL", repoURL)

		listOpts := make([]client.ListOption, 1, 2)
		listOpts[0] = client.MatchingFields{
			indexer.WarehousesBySubscribedURLsField: repoURL,
		}
		if project != "" {
			listOpts = append(listOpts, client.InNamespace(project))
		}

		ws := kargoapi.WarehouseList{}
		if err := c.List(ctx, &ws, listOpts...); err != nil {
			repoLogger.Error(err, "error listing subscribed Warehouses")
			xhttp.WriteErrorJSON(w, err)
			return
		}

		warehouses = append(warehouses, ws.Items...)
	}

	slices.SortFunc(warehouses, func(lhs, rhs kargoapi.Warehouse) int {
		return strings.Compare(lhs.Namespace+lhs.Name, rhs.Namespace+rhs.Name)
	})
	warehouses = slices.CompactFunc(warehouses, func(lhs, rhs kargoapi.Warehouse) bool {
		return lhs.Namespace == rhs.Namespace && lhs.Name == rhs.Name
	})

	logger.Debug("found Warehouses to refresh", "count", len(warehouses))

	var failures int
	for _, wh := range warehouses {
		objKey := client.ObjectKeyFromObject(&wh)
		if _, err := api.RefreshWarehouse(ctx, c, objKey); err != nil {
			logger.Error(err, "error refreshing Warehouse", "objectKey", objKey)
			failures++
		} else {
			logger.Debug("refreshed Warehouse", "objectKey", objKey)
		}
	}

	if failures > 0 {
		xhttp.WriteResponseJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": fmt.Sprintf("failed to refresh %d of %d warehouses",
					failures,
					len(warehouses),
				),
			},
		)
		return
	}
	xhttp.WriteResponseJSON(
		w,
		http.StatusOK,
		map[string]string{
			"msg": fmt.Sprintf("refreshed %d warehouse(s)", len(warehouses)),
		},
	)
}
