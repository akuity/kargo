package replicatedresource

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	libWebhook "github.com/akuity/kargo/pkg/webhook/kubernetes"
)

type webhook struct {
	cfg libWebhook.Config
}

// SetupWebhookWithManager registers the replicated resource validating
// webhook with the given manager.
func SetupWebhookWithManager(
	cfg libWebhook.Config,
	mgr ctrl.Manager,
) error {
	w := newWebhook(cfg)
	mgr.GetWebhookServer().Register(
		"/validate-v1-replicated-resource",
		&admission.Webhook{Handler: w},
	)
	return nil
}

func newWebhook(cfg libWebhook.Config) *webhook {
	return &webhook{cfg: cfg}
}

func (w *webhook) Handle(
	_ context.Context,
	req admission.Request,
) admission.Response {
	if req.UserInfo.Username == w.cfg.ManagementControllerUsername {
		return admission.Allowed("request from Kargo management controller")
	}
	return admission.Denied(
		"replicated resources are managed by Kargo" +
			" and cannot be created, modified, or deleted by end users",
	)
}
