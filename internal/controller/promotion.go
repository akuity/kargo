package controller

import (
	"context"
	"fmt"
	"io/ioutil"
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

func (t *ticketReconciler) promoteImages(
	ctx context.Context,
	ticket *api.Ticket,
	app *argocd.Application,
) (string, error) {
	logger := t.logger.WithFields(log.Fields{})

	// Create a temporary home directory for everything we're about to do
	homeDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", errors.Wrapf(
			err,
			"error creating temporary workspace for cloning repo %q",
			app.Spec.Source.RepoURL,
		)
	}
	defer os.RemoveAll(homeDir)
	t.logger.WithFields(log.Fields{
		"path": homeDir,
	}).Debug("created temporary home directory")

	// Set up auth
	if err = git.SetupAuth(
		ctx,
		app.Spec.Source.RepoURL,
		homeDir,
		t.argoDB,
		logger,
	); err != nil {
		return "", err
	}

	// Clone the repo
	repoDir, err := git.Clone(app.Spec.Source.RepoURL, homeDir, logger)
	if err != nil {
		return "", err
	}

	// TODO: This is hard-coded for now, but there's a possibility here of later
	// supporting other tools and patterns.
	sha, err := t.promotionStrategyRenderedYAMLBranchesWithKustomize(
		ctx,
		ticket,
		app,
		homeDir,
		repoDir,
	)
	if err != nil {
		return "", err
	}

	// Force the Argo CD Application to refresh and sync?
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
func (t *ticketReconciler) promotionStrategyRenderedYAMLBranchesWithKustomize(
	ctx context.Context,
	ticket *api.Ticket,
	app *argocd.Application,
	homeDir string,
	repoDir string,
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
	appDir := filepath.Join(repoDir, app.Spec.Source.TargetRevision)

	// Set the image
	for _, image := range ticket.Change.NewImages.Images {
		if err := kustomize.SetImage(appDir, image, logger); err != nil {
			return "", err
		}
	}

	// Render Application-specific YAML
	// TODO: We may need to buffer this or use a file instead because the rendered
	// YAML could be quite large.
	yamlBytes, err :=
		kustomize.Build(app.Spec.Source.TargetRevision, appDir, logger)
	if err != nil {
		return "", err
	}

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
	cmd := exec.Command("git", "commit", "-am", commitMsg)
	cmd.Dir = repoDir // We need to be in the root of the repo for this
	if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
		return "", errors.Wrap(err, "error committing changes to source branch")
	}
	logger.Debug("committed changes to the source branch")

	// Push the changes to the source branch
	cmd = exec.Command("git", "push", "origin", "HEAD")
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
		return "", errors.Wrap(err, "error pushing changes to source branch")
	}
	logger.Debug("pushed changes to the source branch")

	// Check if the Application-specific branch exists on the remote
	appBranchExists := true
	cmd = exec.Command( // nolint: gosec
		"git",
		"ls-remote",
		"--heads",
		"--exit-code", // Return 2 if not found
		app.Spec.Source.RepoURL,
		app.Spec.Source.TargetRevision,
	)
	// We need to be anywhere in the root of the repo for this
	cmd.Dir = repoDir
	if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
			return "", errors.Wrapf(
				err,
				"error checking for existence of Application-specific branch %q "+
					"from repo %q",
				app.Spec.Source.TargetRevision,
				app.Spec.Source.RepoURL,
			)
		}
		// If we get to here, exit code was 2 and that means the branch doesn't
		// exist
		appBranchExists = false
	}

	if appBranchExists {
		// Switch to the Application-specific branch
		cmd = exec.Command( // nolint: gosec
			"git",
			"checkout",
			app.Spec.Source.TargetRevision,
			// The next line makes it crystal clear to git that we're checking out
			// a branch. We need to do this since we operate under an assumption that
			// the path to the overlay within the repo == the branch name.
			"--",
		)
		cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
		if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking out Application-specific branch %q from repo %q",
				app.Spec.Source.TargetRevision,
				app.Spec.Source.RepoURL,
			)
		}
		logger.Debug(
			"checked out Application-specific branch",
		)
	} else {
		// Create the Application-specific branch
		cmd = exec.Command( // nolint: gosec
			"git",
			"checkout",
			"--orphan",
			app.Spec.Source.TargetRevision,
			// The next line makes it crystal clear to git that we're checking out
			// a branch. We need to do this since we operate under an assumption that
			// the path to the overlay within the repo == the branch name.
			"--",
		)
		cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
		if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
			return "", errors.Wrapf(
				err,
				"error creating orphaned Application-specific branch %q from repo %q",
				app.Spec.Source.TargetRevision,
				app.Spec.Source.RepoURL,
			)
		}
		logger.Debug(
			"created Application-specific branch",
		)
	}

	// Remove existing rendered YAML (or files from the source branch that were
	// left behind when the orphaned Application-specific branch was created)
	files, err := filepath.Glob(filepath.Join(repoDir, "*"))
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
		filepath.Join(repoDir, "all.yaml"),
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
	commitMsg = ""
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
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir // We need to be in the root of the repo for this
	if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
		return "", errors.Wrapf(
			err,
			"error staging changes for commit to Application-specific branch %q",
			app.Spec.Source.TargetRevision,
		)
	}
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = repoDir // We need to be in the root of the repo for this
	if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
		return "", errors.Wrapf(
			err,
			"error committing changes to Application-specific branch %q",
			app.Spec.Source.TargetRevision,
		)
	}
	logger.Debug(
		"committed changes to Application-specific branch",
	)

	// Push the changes to the Application-specific branch
	cmd = exec.Command( // nolint: gosec
		"git",
		"push",
		"origin",
		app.Spec.Source.TargetRevision,
	)
	cmd.Dir = repoDir // We need to be anywhere in the root of the repo for this
	if _, err = git.ExecCommand(cmd, homeDir, logger); err != nil {
		return "", errors.Wrapf(
			err,
			"error pushing changes to Application-specific branch %q",
			app.Spec.Source.TargetRevision,
		)
	}
	logger.Debug(
		"pushed changes to Application-specific branch",
	)

	// Get the ID of the last commit
	sha, err := git.LastCommitID(repoDir)
	if err != nil {
		return "", err
	}
	logger.Debug("obtained sha of commit to Application-specific branch")
	return sha, nil
}
