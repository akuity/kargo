package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	api "github.com/akuityio/k8sta/api/v1alpha1"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuityio/k8sta/internal/git"
	"github.com/akuityio/k8sta/internal/kustomize"
)

// PromotionStrategy is the signature for any function that can promote the
// changes represented by the provided Ticket to the environment represented by
// the provided Argo CD Application by interacting with the provided git
// repository. A RenderStrategy must also be provided by the caller to provide
// integration with config management tools such as kustomize or ytt. Except in
// the event of an error, functions implementing this signature MUST return a
// commit ID (sha). The ticketReconciler will consider a promotion to a given
// environment complete when the commit ID returned from this function is
// visible in the corresponding Argo CD Application's sync history.
type PromotionStrategy func(
	context.Context,
	*api.Ticket,
	*argocd.Application,
	git.Repo,
	RenderStrategy,
) (string, error)

func (t *ticketReconciler) promote(
	ctx context.Context,
	ticket *api.Ticket,
	app *argocd.Application,
) (string, error) {
	logger := t.logger.WithFields(log.Fields{})

	repoCreds, err := getRepoCredentials(ctx, app.Spec.Source.RepoURL, t.argoDB)
	if err != nil {
		return "", err
	}

	repo, err := git.Clone(ctx, app.Spec.Source.RepoURL, repoCreds)
	if err != err {
		return "", err
	}
	defer repo.Close()
	logger.WithFields(log.Fields{
		"url": app.Spec.Source.RepoURL,
	}).Debug("cloned git repository")

	// TODO: For now this is hard-coded to use the rendered YAML branches pattern,
	// but it's possible to later support other approaches by passing a different
	// implementation of the PromotionStrategy function type.
	var promote PromotionStrategy = t.promoteViaRenderedYAMLBranch
	sha, err := promote(
		ctx,
		ticket,
		app,
		repo,
		// TODO: For now this is hard-coded to use kustomize, but it's possible
		// to later support ytt as well by passing a different implementation of
		// the RenderStrategy interface.
		&kustomize.RenderStrategy{},
	)
	if err != nil {
		return "", err
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

	return sha, nil
}

// nolint: gocyclo
func (t *ticketReconciler) promoteViaRenderedYAMLBranch(
	ctx context.Context,
	ticket *api.Ticket,
	app *argocd.Application,
	repo git.Repo,
	renderStrategy RenderStrategy,
) (string, error) {
	logger := t.logger.WithFields(
		log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": app.Spec.Source.TargetRevision,
		},
	)

	// We assume the Application-specific overlay path within the source branch ==
	// the name of the Application-specific branch that the final rendered YAML
	// will live in.
	// TODO: Nothing enforced this assumption yet.
	appDir := filepath.Join(repo.WorkingDir(), app.Spec.Source.TargetRevision)

	// Only do this for image changes
	if ticket.Change.NewImages != nil {
		for _, image := range ticket.Change.NewImages.Images {
			if err := renderStrategy.SetImage(appDir, image); err != nil {
				return "", err
			}
			logger.Debug("set image")
		}
	}

	// Render Application-specific YAML
	baseDir := filepath.Join(repo.WorkingDir(), "base")
	// TODO: We may need to buffer this or use a file instead because the rendered
	// YAML could be quite large.
	yamlBytes, err := renderStrategy.Build(baseDir, appDir)
	if err != nil {
		return "", err
	}
	logger.Debug("built configuration")

	// Only do this for image changes
	if ticket.Change.NewImages != nil {
		// Commit the changes to the source branch
		var commitMsg string
		if len(ticket.Change.NewImages.Images) == 1 {
			commitMsg = fmt.Sprintf(
				"k8sta: updating %s to use image %s:%s",
				app.Spec.Source.TargetRevision,
				ticket.Change.NewImages.Images[0].Repo,
				ticket.Change.NewImages.Images[0].Tag,
			)
		} else {
			commitMsg = "k8sta: updating %s to use new images"
			for _, image := range ticket.Change.NewImages.Images {
				commitMsg = fmt.Sprintf(
					"%s\n * %s:%s",
					commitMsg,
					image.Repo,
					image.Tag,
				)
			}
		}
		if err = repo.AddAllAndCommit(commitMsg); err != nil {
			return "", err
		}
		log.WithFields(log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": "HEAD",
		}).Debug("committed changes")

		// Push the changes to the source branch
		if err = repo.Push(); err != nil {
			return "", err
		}
		logger.WithFields(log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": "HEAD",
		}).Debug("pushed changes")
	}

	// Check if the Application-specific branch exists on the remote
	appBranchExists, err := repo.RemoteBranchExists(
		app.Spec.Source.TargetRevision,
	)
	if err != nil {
		return "", err
	}

	if appBranchExists {
		log.WithFields(log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": app.Spec.Source.TargetRevision,
		}).Debug("branch exists")
		if err = repo.Checkout(
			app.Spec.Source.TargetRevision,
		); err != nil {
			return "", err
		}
		logger.WithFields(log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": app.Spec.Source.TargetRevision,
		}).Debug("checked out branch")
	} else {
		log.WithFields(log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": app.Spec.Source.TargetRevision,
		}).Debug("branch does not exist")
		if err = repo.CreateOrphanedBranch(
			app.Spec.Source.TargetRevision,
		); err != nil {
			return "", err
		}
		logger.WithFields(log.Fields{
			"repo":   app.Spec.Source.RepoURL,
			"branch": app.Spec.Source.TargetRevision,
		}).Debug("created orphaned branch")
	}

	// Remove existing rendered YAML (or files from the source branch that were
	// left behind when the orphaned Application-specific branch was created)
	files, err := filepath.Glob(filepath.Join(repo.WorkingDir(), "*"))
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error listing files in Application-specific branch %q",
			app.Spec.Source.TargetRevision,
		)
	}
	for _, file := range files {
		if _, fileName := filepath.Split(file); fileName == ".git" {
			continue
		}
		if err = os.RemoveAll(file); err != nil {
			return "", errors.Wrapf(
				err,
				"error deleting file %q from Application-specific branch %q",
				file,
				app.Spec.Source.TargetRevision,
			)
		}
	}
	logger.Debug("removed existing rendered YAML")

	// Write the new rendered YAML
	if err = os.WriteFile( // nolint: gosec
		filepath.Join(repo.WorkingDir(), "all.yaml"),
		yamlBytes,
		0644,
	); err != nil {
		return "", errors.Wrapf(
			err,
			"error writing rendered YAML to Application-specific branch %q",
			app.Spec.Source.TargetRevision,
		)
	}
	logger.Debug("wrote new rendered YAML")

	// Commit the changes to the Application-specific branch
	var commitMsg string
	if ticket.Change.NewImages != nil {
		if len(ticket.Change.NewImages.Images) == 1 {
			commitMsg = fmt.Sprintf(
				"k8sta: updating to use new image %s:%s",
				ticket.Change.NewImages.Images[0].Repo,
				ticket.Change.NewImages.Images[0].Tag,
			)
		} else {
			commitMsg = "k8sta: updating to use new images"
			for _, image := range ticket.Change.NewImages.Images {
				commitMsg = fmt.Sprintf(
					"%s\n * %s:%s",
					commitMsg,
					image.Repo,
					image.Tag,
				)
			}
		}
	} else {
		commitMsg = fmt.Sprintf(
			"k8sta: updating with base configuration changes from %s",
			ticket.Change.BaseConfiguration.Commit,
		)
	}
	if err = repo.AddAllAndCommit(commitMsg); err != nil {
		return "", err
	}
	log.WithFields(log.Fields{
		"repo":   app.Spec.Source.RepoURL,
		"branch": app.Spec.Source.TargetRevision,
	}).Debug("committed changes")

	// Push the changes to the Application-specific branch
	if err = repo.Push(); err != nil {
		return "", err
	}
	logger.WithFields(log.Fields{
		"repo":   app.Spec.Source.RepoURL,
		"branch": app.Spec.Source.TargetRevision,
	}).Debug("pushed changes")

	// Get the ID of the last commit
	sha, err := repo.LastCommitID()
	if err != nil {
		return "", err
	}
	logger.Debug("obtained sha of commit to Application-specific branch")
	return sha, nil
}
