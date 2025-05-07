package handlers

import (
	"encoding/json"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/webhook/external/providers"
)

// NewRefreshWarehouseWebhook handles push events for the designated provider.
// After the provider has been resolved and the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func (f *Factory) NewRefreshWarehouseWebhook(name providers.Name) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.log.Info("initializing provider")
		p, err := providers.New(name)
		if err != nil {
			// If there's a missing secret/env var
			// we don't want to leak that information
			// from our API so just log the part that's
			// useful to us and return a generic error
			// to the client.
			f.log.Error(err, "failed to initialize provider")
			http.Error(w,
				"failed to initialize provider",
				http.StatusInternalServerError,
			)
			return
		}

		f.log.Info("provider initialized",
			"provider", name.String(),
		)

		f.log.Info("authenticating request")
		if err = p.Authenticate(r); err != nil {
			f.log.Error(err, "failed to authenticate request")
			http.Error(w,
				"failed to authenticate request",
				http.StatusUnauthorized,
			)
			return
		}
		f.log.Info("request authenticated")

		f.log.Info("retrieving source repo")
		repo, err := p.Repository(r)
		if err != nil {
			f.log.Error(err, "failed to retrieve source repo")
			http.Error(w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		f.log.Info("repo retrieved", "name", repo)

		ctx := r.Context()
		var warehouses v1alpha1.WarehouseList
		err = f.Client.List(
			ctx,
			&warehouses,
			client.MatchingFields{
				indexer.WarehousesBySubscribedURLsField: repo,
			},
		)
		if err != nil {
			f.log.Error(err, "failed to list warehouses")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		f.log.Info("listed warehouses",
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
				f.Client,
				types.NamespacedName{
					Namespace: wh.GetNamespace(),
					Name:      wh.GetName(),
				},
			)
			if err != nil {
				f.log.Error(err, "failed to patch annotations",
					"warehouse", wh.GetName(),
				)
				resp.Errors[wh.GetName()] = err.Error()
				resp.WarehousesFailedToRefresh++
			} else {
				f.log.Info("successfully patched annotations",
					"warehouse", wh.GetName(),
				)
				resp.WarehousesSuccessfullyRefreshed++
			}
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
