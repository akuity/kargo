package handlers

import (
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/webhook/external/providers"
)

// NewRefreshWarehouseWebhook handles push events for the designated provider.
// After the provider has been resolved and the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func NewRefreshWarehouseWebhook(p providers.Provider, c client.Client) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		repo, err := p.GetRepository(r)
		if err != nil {
			code := xhttp.CodeFrom(err)
			xhttp.WriteErrorf(w,
				code,
				"failed to get repository: %w",
				err,
			)
			return
		}
		logger.Debug("source repository retrieved", "name", repo)

		var warehouses v1alpha1.WarehouseList
		err = c.List(
			ctx,
			&warehouses,
			client.MatchingFields{
				indexer.WarehousesBySubscribedURLsField: repo,
			},
		)
		if err != nil {
			logger.Error(err, "failed to list warehouses")
			xhttp.WriteServerErrorf(w,
				"failed to list warehouses: %w",
				err,
			)
			return
		}

		logger.Debug("listed warehouses",
			"num-warehouses", len(warehouses.Items),
		)

		var numSuccessfullyRefreshed, numRefreshFailures int
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
				)
				numRefreshFailures++
			} else {
				logger.Debug("successfully patched annotations",
					"warehouse", wh.GetName(),
				)
				numSuccessfullyRefreshed++
			}
		}

		logger.Debug("execution complete",
			"num-successful-refreshes", numSuccessfullyRefreshed,
			"num-refresh-failures", numRefreshFailures,
		)

		if numRefreshFailures > 0 {
			xhttp.WriteServerErrorf(w,
				"failed to refresh %d of %d warehouses",
				numRefreshFailures,
				len(warehouses.Items),
			)
			return
		}

		xhttp.Writef(w,
			http.StatusOK,
			"refreshed %d warehouses",
			len(warehouses.Items),
		)
	})
}
