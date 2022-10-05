package controller

import (
	"context"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuityio/k8sta/internal/bookkeeper"
)

func (t *ticketReconciler) promote(
	ctx context.Context,
	track *api.Track,
	ticket *api.Ticket,
	app *argocd.Application,
) (string, error) {
	repoCreds, err := getRepoCredentials(ctx, app.Spec.Source.RepoURL, t.argoDB)
	if err != nil {
		return "", err
	}

	// Call the Bookkeeping service
	renderReq := bookkeeper.RenderRequest{
		RepoURL:          app.Spec.Source.RepoURL,
		RepoCreds:        repoCreds,
		TargetBranch:     app.Spec.Source.TargetRevision,
		ConfigManagement: track.Spec.ConfigManagement,
	}
	var res bookkeeper.Response
	if ticket.Change.BaseConfiguration != nil {
		res, err = t.bookkeeperService.RenderConfig(ctx, renderReq)
	} else {
		res, err = t.bookkeeperService.UpdateImage(
			ctx,
			bookkeeper.ImageUpdateRequest{
				RenderRequest: renderReq,
				Images:        ticket.Change.NewImages.Images,
			},
		)
	}
	if err != nil {
		return "", errors.Wrapf(err, "bookkeeping error")
	}

	// Force the Argo CD Application to refresh and sync
	patch := client.MergeFrom(app.DeepCopy())
	app.ObjectMeta.Annotations[argocd.AnnotationKeyRefresh] =
		string(argocd.RefreshTypeHard)
	app.Operation = &argocd.Operation{
		Sync: &argocd.SyncOperation{
			Revision: app.Spec.Source.TargetRevision,
		},
	}
	if err = t.client.Patch(ctx, app, patch, &client.PatchOptions{}); err != nil {
		return "", errors.Wrapf(
			err,
			"error patching Argo CD Application %q to coerce refresh and sync",
			app.Name,
		)
	}
	t.logger.WithFields(log.Fields{
		"app": app.Name,
	}).Debug("triggered refresh of Argo CD Application")

	return res.CommitID, nil
}
