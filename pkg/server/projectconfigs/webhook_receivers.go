package projectconfigs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

// RegisterSystem adds the routes that span Projects to the system route
// group. Unlike the routes Register adds, these are not scoped to a Project,
// so each checks cluster-wide access itself.
func (h *Handler) RegisterSystem(system *gin.RouterGroup) {
	system.GET("/webhook-receivers", h.lookupWebhookReceiver)
}

// @id LookupWebhookReceiver
// @Summary Look up a webhook receiver by path
// @Description Find the webhook receiver that owns a path, and the Project
// @Description whose ProjectConfig configures it. Only the external webhooks
// @Description server is meant to call this; it requires list on
// @Description projectconfigs across the cluster.
// @Tags System, Config
// @Security BearerAuth
// @Produce json
// @Param path query string true "Webhook receiver path"
// @Success 200 {object} wire.WebhookReceiverLookup "Webhook receiver"
// @Router /v1beta1/system/webhook-receivers [get]
func (h *Handler) lookupWebhookReceiver(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.authorize(ctx, can.List().Kargo("projectconfigs")); err != nil {
		_ = c.Error(err)
		return
	}
	path := c.Query("path")
	if path == "" {
		_ = c.Error(libhttp.Error(
			errors.New("the path query parameter is required"),
			http.StatusBadRequest,
		))
		return
	}
	cfg, err := h.store.GetByWebhookReceiverPath(ctx, path)
	if err != nil {
		_ = c.Error(storeError(err, path))
		return
	}
	var details *kargoapi.WebhookReceiverDetails
	for _, r := range cfg.Receivers {
		if r.Path == path {
			details = &kargoapi.WebhookReceiverDetails{
				Name: r.Name,
				Path: r.Path,
				URL:  r.URL,
			}
			break
		}
	}
	if details != nil {
		for _, rc := range projectconfigstore.ToAPI(cfg).Spec.WebhookReceivers {
			if rc.Name == details.Name {
				c.JSON(http.StatusOK, wire.WebhookReceiverLookup{
					Project:  cfg.ProjectName,
					Config:   rc,
					Receiver: *details,
				})
				return
			}
		}
	}
	// The spec no longer has the receiver the path was recorded for, so the
	// path is about to stop resolving.
	_ = c.Error(apierrors.NewNotFound(
		kargoapi.GroupVersion.WithResource("projectconfigs").GroupResource(),
		path,
	))
}
