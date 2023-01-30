package controller

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/akuityio/bookkeeper"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/kustomize"
	"github.com/akuityio/kargo/internal/yaml"
)

func (e *environmentReconciler) promote(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	if env.Spec.PromotionMechanisms == nil {
		return newState,
			errors.New("spec contains insufficient instructions to reach new state")
	}

	var err error
	if env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper != nil {
		if newState, err = e.promoteWithBookkeeper(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Bookkeeper")
		}
	} else if env.Spec.PromotionMechanisms.ConfigManagement.Kustomize != nil {
		if newState, err = e.promoteWithKustomize(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Kustomize")
		}
	} else if env.Spec.PromotionMechanisms.ConfigManagement.Helm != nil {
		if newState, err = e.promoteWithHelm(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Helm")
		}
	}

	if env.Spec.PromotionMechanisms.ArgoCD != nil {
		if err = e.promoteWithArgoCD(ctx, env, newState); err != nil {
			return newState, errors.Wrap(err, "error promoting via Argo CD")
		}
	}

	e.logger.WithFields(log.Fields{
		"namespace": env.Namespace,
		"name":      env.Name,
		"state":     newState.ID,
		"git":       newState.GitCommit,
		"images":    newState.Images,
	}).Debug("completed promotion")

	return newState, nil
}

func (e *environmentReconciler) promoteWithBookkeeper(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	// If any of the following is true, this function ought not to have been
	// invoked, but we don't take that on faith.
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper.TargetBranch == "" { // nolint: lll
		return newState, nil
	}

	images := make([]string, len(newState.Images))
	for i, image := range newState.Images {
		images[i] = fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
	}
	creds, err := e.getGitRepoCredentialsFn(ctx, newState.GitCommit.RepoURL)
	if err != nil {
		return newState, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			newState.GitCommit.RepoURL,
		)
	}
	req := bookkeeper.RenderRequest{
		RepoURL: newState.GitCommit.RepoURL,
		RepoCreds: bookkeeper.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		},
		Commit:       newState.GitCommit.ID,
		Images:       images,
		TargetBranch: env.Spec.PromotionMechanisms.ConfigManagement.Bookkeeper.TargetBranch, // nolint: lll
	}
	res, err := e.renderManifestsWithBookkeeperFn(ctx, req)
	if err != nil {
		return newState,
			errors.Wrap(err, "error rendering manifests via Bookkeeper")
	}

	if res.ActionTaken == bookkeeper.ActionTakenPushedDirectly ||
		res.ActionTaken == bookkeeper.ActionTakenNone {
		newState.HealthCheckCommit = res.CommitID
	}
	// TODO: This is a fairly large outstanding question. How do we deal with PRs?
	// When a PR is opened, we don't immediately know the

	return newState, nil
}

// TODO: Add some logging to this function
// nolint: gocyclo
func (e *environmentReconciler) promoteWithKustomize(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement.Kustomize == nil ||
		len(env.Spec.PromotionMechanisms.ConfigManagement.Kustomize.Images) == 0 ||
		len(newState.Images) == 0 {
		return newState, nil
	}

	if env.Spec.GitRepo == nil || env.Spec.GitRepo.URL == "" {
		return newState, errors.New(
			"cannot promote images via Kustomize because spec does not contain " +
				"git repo details",
		)
	}

	repoURL := env.Spec.GitRepo.URL

	creds, err := e.getGitRepoCredentialsFn(ctx, repoURL)
	if err != nil {
		return newState, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			repoURL,
		)
	}

	repo, err := e.gitCloneFn(
		ctx,
		env.Spec.GitRepo.URL,
		git.RepoCredentials{
			Username: creds.Username,
			Password: creds.Password,
		},
	)
	if err != nil {
		return newState, errors.Wrapf(err, "error cloning git repo %q", repoURL)
	}
	if repo != nil { // This could be nil during a test
		defer repo.Close()
	}
	logger := e.logger.WithFields(log.Fields{
		"environment": env.Name,
		"namespace":   env.Namespace,
		"repoURL":     repoURL,
	})
	logger.Debug("cloned git repo")

	branch := env.Spec.GitRepo.Branch

	if branch != "" {
		if err = e.checkoutBranchFn(repo, branch); err != nil {
			return newState, errors.Wrapf(
				err,
				"error checking out branch %q from git repo",
				repoURL,
			)
		}
	}
	logger = logger.WithField("branch", branch)
	logger.Debug("checked out branch")

	imgUpdates := env.Spec.PromotionMechanisms.ConfigManagement.Kustomize.Images
	for _, imgUpdate := range imgUpdates {
		var tag string
		for _, img := range newState.Images {
			if img.RepoURL == imgUpdate.Image {
				tag = img.Tag
				break
			}
		}
		if tag == "" {
			// TODO: Warn?
			continue
		}
		dir := filepath.Join(repo.WorkingDir(), imgUpdate.Path)
		if err = kustomize.SetImage(dir, imgUpdate.Image, tag); err != nil {
			return newState, errors.Wrapf(
				err,
				"error updating image %q to tag %q using Kustomize",
				imgUpdate.Image,
				tag,
			)
		}
	}

	var hasDiffs bool
	if hasDiffs, err = repo.HasDiffs(); err != nil {
		return newState, errors.Wrap(err, "error checking for diffs")
	} else if !hasDiffs {
		// We only want health checks to factor in a specific commit if we subscribe
		// to the Git repo. If we don't subscribe to the Git repo, we're probably in
		// a case where the associated Application resources tracks the head of a
		// branch and we don't want to count Applications as unhealthy just on
		// account of (with no Kargo involvement) having moved on to a newer commit
		// at the head of that branch.
		//
		// TODO: This seems correct for zero environment, but it might not hold up
		// for non-zero environments.
		if env.Spec.Subscriptions != nil &&
			env.Spec.Subscriptions.Repos != nil &&
			env.Spec.Subscriptions.Repos.Git {
			newState.HealthCheckCommit, err = repo.LastCommitID()
		}
		return newState, errors.Wrap(err, "error getting last commit ID")
	}

	if err = repo.AddAllAndCommit("updating images"); err != nil {
		return newState,
			errors.Wrap(err, "error committing updated images to git repo")
	}

	if err = repo.Push(); err != nil {
		return newState, errors.Wrap(
			err,
			"error pushing commit containing updated images to git repo",
		)
	}

	// We only want health checks to factor in a specific commit if we subscribe
	// to the Git repo. If we don't subscribe to the Git repo, we're probably in
	// a case where the associated Application resources track the head of a
	// branch and we don't want to count Applications as unhealthy just on
	// account of (with no Kargo involvement) having moved on to a newer commit
	// at the head of that branch.
	//
	// TODO: This seems correct for zero environment, but it might not hold up
	// for non-zero environments.
	if env.Spec.Subscriptions != nil &&
		env.Spec.Subscriptions.Repos != nil &&
		env.Spec.Subscriptions.Repos.Git {
		newState.HealthCheckCommit, err = repo.LastCommitID()
	}
	return newState, errors.Wrap(err, "error getting last commit ID")
}

// TODO: Add some logging to this function
// nolint: gocyclo
func (e *environmentReconciler) promoteWithHelm(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement == nil ||
		env.Spec.PromotionMechanisms.ConfigManagement.Helm == nil ||
		len(env.Spec.PromotionMechanisms.ConfigManagement.Helm.Images) == 0 ||
		len(newState.Images) == 0 {
		return newState, nil
	}

	if env.Spec.GitRepo == nil || env.Spec.GitRepo.URL == "" {
		return newState, errors.New(
			"cannot promote images via Helm because spec does not contain " +
				"git repo details",
		)
	}

	repoURL := env.Spec.GitRepo.URL

	creds, err := e.getGitRepoCredentialsFn(ctx, repoURL)
	if err != nil {
		return newState, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			repoURL,
		)
	}

	repo, err := e.gitCloneFn(
		ctx,
		env.Spec.GitRepo.URL,
		git.RepoCredentials{
			Username: creds.Username,
			Password: creds.Password,
		},
	)
	if err != nil {
		return newState, errors.Wrapf(err, "error cloning git repo %q", repoURL)
	}
	if repo != nil { // This could be nil during a test
		defer repo.Close()
	}
	logger := e.logger.WithFields(log.Fields{
		"environment": env.Name,
		"namespace":   env.Namespace,
		"repoURL":     repoURL,
	})
	logger.Debug("cloned git repo")

	branch := env.Spec.GitRepo.Branch

	if branch != "" {
		if err = e.checkoutBranchFn(repo, branch); err != nil {
			return newState, errors.Wrapf(
				err,
				"error checking out branch %q from git repo",
				repoURL,
			)
		}
	}
	logger = logger.WithField("branch", branch)
	logger.Debug("checked out branch")

	imgUpdates := env.Spec.PromotionMechanisms.ConfigManagement.Helm.Images
	changesByFile := buildChangeMapsByFile(newState.Images, imgUpdates)
	for file, changes := range changesByFile {
		if err = yaml.SetStringsInFile(
			filepath.Join(repo.WorkingDir(), file),
			changes,
		); err != nil {
			return newState, errors.Wrapf(
				err,
				"error updating values in file %q",
				file,
			)
		}
	}

	var hasDiffs bool
	if hasDiffs, err = repo.HasDiffs(); err != nil {
		return newState, errors.Wrap(err, "error checking for diffs")
	} else if !hasDiffs {
		// We only want health checks to factor in a specific commit if we subscribe
		// to the Git repo. If we don't subscribe to the Git repo, we're probably in
		// a case where the associated Application resources tracks the head of a
		// branch and we don't want to count Applications as unhealthy just on
		// account of (with no Kargo involvement) having moved on to a newer commit
		// at the head of that branch.
		//
		// TODO: This seems correct for zero environment, but it might not hold up
		// for non-zero environments.
		if env.Spec.Subscriptions != nil &&
			env.Spec.Subscriptions.Repos != nil &&
			env.Spec.Subscriptions.Repos.Git {
			newState.HealthCheckCommit, err = repo.LastCommitID()
		}
		return newState, errors.Wrap(err, "error getting last commit ID")
	}

	if err = repo.AddAllAndCommit("updating images"); err != nil {
		return newState,
			errors.Wrap(err, "error committing updated images to git repo")
	}

	if err = repo.Push(); err != nil {
		return newState, errors.Wrap(
			err,
			"error pushing commit containing updated images to git repo",
		)
	}

	// We only want health checks to factor in a specific commit if we subscribe
	// to the Git repo. If we don't subscribe to the Git repo, we're probably in
	// a case where the associated Application resources track the head of a
	// branch and we don't want to count Applications as unhealthy just on
	// account of (with no Kargo involvement) having moved on to a newer commit
	// at the head of that branch.
	//
	// TODO: This seems correct for zero environment, but it might not hold up
	// for non-zero environments.
	if env.Spec.Subscriptions != nil &&
		env.Spec.Subscriptions.Repos != nil &&
		env.Spec.Subscriptions.Repos.Git {
		newState.HealthCheckCommit, err = repo.LastCommitID()
	}
	return newState, errors.Wrap(err, "error getting last commit ID")
}

// buildChangeMapsByFile takes a list of images and a list of instructions about
// changes that should be made to various YAML files and distills them to a
// map of maps that indexes new values for each YAML file by file name and key.
func buildChangeMapsByFile(
	images []api.Image,
	imageUpdates []api.HelmImageUpdate,
) map[string]map[string]string {
	tagsByImage := map[string]string{}
	for _, image := range images {
		tagsByImage[image.RepoURL] = image.Tag
	}

	changesByFile := map[string]map[string]string{}
	for _, imageUpdate := range imageUpdates {
		if imageUpdate.Value != "Image" && imageUpdate.Value != "Tag" {
			// This really shouldn't happen, so we'll ignore it.
			continue
		}
		tag, found := tagsByImage[imageUpdate.Image]
		if !found {
			// There's no change to make in this case.
			continue
		}
		if _, found = changesByFile[imageUpdate.ValuesFilePath]; !found {
			changesByFile[imageUpdate.ValuesFilePath] = map[string]string{}
		}
		if imageUpdate.Value == "Image" {
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] =
				fmt.Sprintf("%s:%s", imageUpdate.Image, tag)
		} else {
			changesByFile[imageUpdate.ValuesFilePath][imageUpdate.Key] = tag
		}
	}

	return changesByFile
}

func (e *environmentReconciler) promoteWithArgoCD(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
) error {
	// If any of the following is true, this function ought not to have been
	// invoked, but we don't take that on faith.
	if env == nil ||
		env.Spec.PromotionMechanisms == nil ||
		env.Spec.PromotionMechanisms.ArgoCD == nil ||
		len(env.Spec.PromotionMechanisms.ArgoCD.AppUpdates) == 0 {
		return nil
	}

	for _, appUpdate := range env.Spec.PromotionMechanisms.ArgoCD.AppUpdates {
		if appUpdate.UpdateTargetRevision && newState.GitCommit != nil {
			if err := e.updateArgoCDAppTargetRevisionFn(
				ctx,
				env.Namespace,
				appUpdate.Name,
				newState.GitCommit.ID,
			); err != nil {
				return errors.Wrapf(
					err,
					"error updating target revision for Argo CD Application %q in "+
						"namespace %q",
					appUpdate.Name,
					env.Namespace,
				)
			}
			continue
		}
		if appUpdate.RefreshAndSync {
			if err := e.refreshAndSyncArgoCDAppFn(
				ctx,
				env.Namespace,
				appUpdate.Name,
			); err != nil {
				return errors.Wrapf(
					err,
					"error syncing Argo CD Application %q in namespace %q",
					appUpdate.Name,
					env.Namespace,
				)
			}
		}
	}

	return nil
}
