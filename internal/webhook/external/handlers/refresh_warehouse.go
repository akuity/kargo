package handlers

import (
	"encoding/json"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
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
		logger := logging.LoggerFromContext(r.Context())
		logger.Debug("authenticating request")
		if err := p.Authenticate(r); err != nil {
			logger.Error(err, "failed to authenticate request")
			http.Error(w,
				"failed to authenticate request",
				http.StatusUnauthorized,
			)
			return
		}
		logger.Debug("request authenticated")

		logger.Debug("retrieving source repo")
		repo, err := p.Repository(r)
		if err != nil {
			logger.Error(err, "failed to retrieve source repo")
			http.Error(w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		logger.Info("repo retrieved", "name", repo)

		ctx := r.Context()
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Debug("listed warehouses",
			"num-warehouses", len(warehouses.Items),
		)

		resp := &struct {
			WarehousesSuccessfullyRefreshed int               `json:"warehouses_successfully_refreshed"`
			WarehousesFailedToRefresh       int               `json:"warehouses_failed_to_refresh"`
			Errors                          map[string]string `json:"errors,omitempty"`
		}{
			Errors: make(map[string]string),
		}

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
				logger.Error(err, "failed to patch annotations",
					"warehouse", wh.GetName(),
				)
				resp.Errors[wh.GetName()] = err.Error()
				resp.WarehousesFailedToRefresh++
			} else {
				logger.Debug("successfully patched annotations",
					"warehouse", wh.GetName(),
				)
				resp.WarehousesSuccessfullyRefreshed++
			}
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
