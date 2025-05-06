package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/webhook/external/providers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRefreshWarehouseWebhook handles push events for the designated provider.
// After the provider has been resolved and the request has been authenticated,
// the kubeclient is queried for all warehouses that contain a subscription
// to the repo in question. Those warehouses are then patched with a special
// annotation that signals down stream logic to refresh the warehouse.
func NewRefreshWarehouseWebhook(name providers.Name, l *slog.Logger, c client.Client) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Info("initializing provider")
		p, err := providers.New(name)
		if err != nil {
			// If there's a missing secret/env var
			// we don't want to leak that information
			// from our API so just log the part that's
			// useful to us and return a generic error
			// to the client.
			l.Error(
				"failed to initialize provider",
				"error", err.Error(),
			)
			http.Error(w,
				"failed to initialize provider",
				http.StatusInternalServerError,
			)
			return
		}

		l.Info("provider initialized",
			"provider", name.String(),
		)

		l.Info("authenticating request")
		if err := p.Authenticate(r); err != nil {
			l.Error(
				"failed to authenticate request",
				"error", err.Error(),
			)
			http.Error(w,
				"failed to authenticate request",
				http.StatusUnauthorized,
			)
			return
		}
		l.Info("request authenticated")

		l.Info("retrieving event")
		event, err := p.Event(r)
		if err != nil {
			l.Error("failed to retrieve event", "error", err.Error())
			http.Error(w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		l.Info("event retrieved",
			"commit", event.Commit(),
			"pushed-by", event.PushedBy(),
			"repository", event.Repository(),
		)

		ctx := r.Context()
		var warehouses v1alpha1.WarehouseList
		err = c.List(
			ctx,
			&warehouses,
			// TODO(Faris): Merge https://github.com/akuity/kargo/pull/3969
			// client.MatchingFields{
			// 	indexer.WarehouseRepoURLIndexKey: event.Repository(),
			// },
		)
		if err != nil {
			l.Error("failed to list warehouses", "error", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		l.Info("listed warehouses",
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
			err = api.PatchAnnotation(
				ctx,
				c,
				&wh,
				v1alpha1.AnnotationKeyRefresh,
				time.Now().Format(time.RFC3339),
			)
			if err != nil {
				l.Error("failed to patch annotations",
					"warehouse", wh.GetName(),
					"error", err.Error(),
				)
				resp.Errors[wh.GetName()] = err.Error()
				resp.WarehousesFailedToRefresh++
			} else {
				l.Info("successfully patched annotations",
					"warehouse", wh.GetName(),
				)
				resp.WarehousesSuccessfullyRefreshed++
			}
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
