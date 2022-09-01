package controller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
		kustomize.RenderStrategy,
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
	appBranch := app.Spec.Source.TargetRevision

	logger := t.logger.WithFields(
		log.Fields{
			"ticket":    ticket.Name,
			"namespace": ticket.Namespace,
			"repo":      app.Spec.Source.RepoURL,
			"appBranch": appBranch,
		},
	)

	baseDir := filepath.Join(repo.WorkingDir(), "base")
	appDir := filepath.Join(repo.WorkingDir(), appBranch)
	k8staDir := filepath.Join(repo.WorkingDir(), ".k8sta")
	k8staAppDir := filepath.Join(k8staDir, appBranch)

	if err := kustomize.EnsurePrerenderDir(k8staAppDir); err != nil {
		return "", errors.Wrapf(
			err,
			"error setting up pre-render directory %q",
			k8staAppDir,
		)
	}

	var commitMsg string
	if ticket.Change.NewImages != nil {
		// TODO: We can maybe break this out into its own function.
		//
		// For image only changes, we need to call kustomize.SetImage.
		for _, image := range ticket.Change.NewImages.Images {
			if err := kustomize.SetImage(
				k8staAppDir,
				image,
			); err != nil {
				return "", errors.Wrapf(
					err,
					"error setting image in pre-render directory %q",
					k8staAppDir,
				)
			}
			logger.Debug("set image")
		}
		if len(ticket.Change.NewImages.Images) == 1 {
			commitMsg = fmt.Sprintf(
				"k8sta: updating %s to use image %s:%s",
				appBranch,
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
	} else {
		commitMsg = fmt.Sprintf(
			"k8sta: pre-rendering base configuration changes from %s",
			ticket.Change.BaseConfiguration.Commit,
		)
	}

	// TODO: We can maybe break this out into its own function.
	//
	// For ALL types of changes, We need to use the user's preferred config
	// management tool to complete the first phase of rendering aka
	// "pre-rendering." We save that in the source branch. Later, we take it the
	// "last mile" with kustomize and write that FULLY-rendered config to the
	// Application-specific branch.
	//
	// TODO: We may need to buffer this or use a file instead because the
	// rendered config could be quite large.
	//
	// TODO: Fix this hard-coded placeholder release name -- actually it's only
	// used by the helm strategy.
	yamlBytes, err := renderStrategy("k8sta-demo", baseDir, appDir)
	if err != nil {
		return "", errors.Wrap(
			err,
			"error pre-rendering configuration",
		)
	}
	logger.Debug("pre-rendered configuration")
	// Write/overwrite the pre-rendered config
	if err = os.WriteFile( // nolint: gosec
		filepath.Join(k8staAppDir, "all.yaml"),
		yamlBytes,
		0644,
	); err != nil {
		return "", errors.Wrap(
			err,
			"error writing pre-rendered configuration to source branch",
		)
	}
	logger.Debug("wrote pre-rendered configuration to source branch")

	// Commit pre-rendered config to the local source branch
	if err = repo.AddAllAndCommit(commitMsg); err != nil {
		return "", errors.Wrap(
			err,
			"error committing changes to source branch",
		)
	}
	logger.Debug("committed changes to source branch")

	// Push the changes to the remote source branch
	if err = repo.Push(); err != nil {
		return "", errors.Wrap(
			err,
			"error pushing changes to source branch",
		)
	}
	logger.Debug("pushed changes to the source branch")

	// Now take everything the last mile with kustomize and write the
	// fully-rendered config to an Application-specific branch...

	// Last mile rendering
	//
	// TODO: We may need to buffer this or use a file instead because the
	// rendered config could be quite large.
	cmd := exec.Command("kustomize", "build")
	cmd.Dir = k8staAppDir
	if yamlBytes, err = cmd.Output(); err != nil {
		return "", errors.Wrapf(
			err,
			"error producing fully-rendered configuration: "+
				"error running `%s` in directory %q",
			cmd.String(),
			cmd.Dir,
		)
	}

	// Check if the Application-specific branch exists on the remote
	//
	// TODO: We can break this out into its own function that ensures the
	// existence of a branch.
	var appBranchExists bool
	if appBranchExists, err = repo.RemoteBranchExists(appBranch); err != nil {
		return "", errors.Wrapf(
			err,
			"error checking for existence of remote Application-specific branch %q",
			appBranch,
		)
	} else if appBranchExists {
		logger.Debug("Application-specific branch exists")
		if err = repo.Checkout(appBranch); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking out Application-specific branch %q",
				appBranch,
			)
		}
		logger.Debug("checked out Application-specific branch")
	} else {
		logger.Debug("Application-specific branch does not exist")
		if err = repo.CreateOrphanedBranch(appBranch); err != nil {
			return "", err
		}
		logger.Debug("created orphaned Application-specific branch branch")
	}

	// Remove existing rendered YAML (or files from the source branch that were
	// left behind when the orphaned Application-specific branch was created)
	//
	// TODO: We can break this out into its own function
	files, err := filepath.Glob(filepath.Join(repo.WorkingDir(), "*"))
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error listing files in Application-specific branch %q",
			appBranch,
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
				appBranch,
			)
		}
	}
	logger.Debug("removed existing rendered YAML")

	// Write the new fully-rendered config to the root of the repo
	if err = os.WriteFile( // nolint: gosec
		filepath.Join(repo.WorkingDir(), "all.yaml"),
		yamlBytes,
		0644,
	); err != nil {
		return "", errors.Wrapf(
			err,
			"error writing fully-rendered configuration to Application-specific "+
				"branch %q",
			appBranch,
		)
	}
	logger.Debug("wrote new rendered YAML")

	// Commit the changes to the Application-specific branch
	if ticket.Change.BaseConfiguration != nil {
		commitMsg = fmt.Sprintf(
			"k8sta: updating with base configuration changes from %s",
			ticket.Change.BaseConfiguration.Commit,
		)
	}
	if err = repo.AddAllAndCommit(commitMsg); err != nil {
		return "", errors.Wrapf(
			err,
			"error committing changes to Application-specific branch %q",
			appBranch,
		)
	}
	log.Debug("committed changes to Application-specific branch")

	// Push the changes to the remote Application-specific branch
	if err = repo.Push(); err != nil {
		return "", errors.Wrapf(
			err,
			"error pushing changes to Application-specific branch %q",
			appBranch,
		)
	}
	logger.Debug("pushed changes")

	// Get the ID of the last commit
	sha, err := repo.LastCommitID()
	if err != nil {
		return "", err
	}
	logger.Debug("obtained sha of commit to Application-specific branch")
	return sha, nil
}
