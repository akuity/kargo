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
func NewRefreshWarehouseWebhook(p providers.Provider, log *logging.Logger, kClient client.Client) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("authenticating request")
		if err := p.Authenticate(r); err != nil {
			log.Error(err, "failed to authenticate request")
			http.Error(w,
				"failed to authenticate request",
				http.StatusUnauthorized,
			)
			return
		}
		log.Info("request authenticated")

		log.Info("retrieving source repo")
		repo, err := p.Repository(r)
		if err != nil {
			log.Error(err, "failed to retrieve source repo")
			http.Error(w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		log.Info("repo retrieved", "name", repo)

		ctx := r.Context()
		var warehouses v1alpha1.WarehouseList
		err = kClient.List(
			ctx,
			&warehouses,
			client.MatchingFields{
				indexer.WarehousesBySubscribedURLsField: repo,
			},
		)
		if err != nil {
			log.Error(err, "failed to list warehouses")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("listed warehouses",
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
				kClient,
				types.NamespacedName{
					Namespace: wh.GetNamespace(),
					Name:      wh.GetName(),
				},
			)
			if err != nil {
				log.Error(err, "failed to patch annotations",
					"warehouse", wh.GetName(),
				)
				resp.Errors[wh.GetName()] = err.Error()
				resp.WarehousesFailedToRefresh++
			} else {
				log.Info("successfully patched annotations",
					"warehouse", wh.GetName(),
				)
				resp.WarehousesSuccessfullyRefreshed++
			}
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
