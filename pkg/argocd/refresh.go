package argocd

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// RequestAppRefresh annotates the Argo CD Application with a hard refresh
// request. This ensures Argo CD invalidates its cache and persists the full
// status (including reconciledAt) on the next reconciliation — even when
// health and sync status are unchanged from before the operation.
func RequestAppRefresh(
	ctx context.Context,
	argocdClient client.Client,
	app *argocd.Application,
) {
	patch := client.MergeFrom(app.DeepCopy())
	if app.Annotations == nil {
		app.Annotations = make(map[string]string, 1)
	}
	app.Annotations[argocd.AnnotationKeyRefresh] = string(argocd.RefreshTypeHard)
	if err := argocdClient.Patch(ctx, app, patch); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err,
			"failed to request hard refresh for Argo CD Application",
			"namespace", app.Namespace,
			"name", app.Name,
		)
	}
}
