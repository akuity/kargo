package controller

import (
	"context"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/git"
)

func (e *environmentReconciler) promoteWithGit(
	ctx context.Context,
	env *api.Environment,
	newState api.EnvironmentState,
	commitMsg string,
	cfgMgmtCallback func(git.Repo) error,
) (api.EnvironmentState, error) {
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

	if err = cfgMgmtCallback(repo); err != nil {
		return newState, err
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
		if newState.GitCommit != nil {
			newState.GitCommit.ID, err = repo.LastCommitID()
			newState.HealthCheckCommit = newState.GitCommit.ID
		}
		return newState, errors.Wrap(err, "error getting last commit ID")
	}

	if err = repo.AddAllAndCommit(commitMsg); err != nil {
		return newState,
			errors.Wrap(err, "error committing updates to git repo")
	}

	if err = repo.Push(); err != nil {
		return newState, errors.Wrap(err, "error pushing updates to git repo")
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
	if newState.GitCommit != nil {
		newState.GitCommit.ID, err = repo.LastCommitID()
		newState.HealthCheckCommit = newState.GitCommit.ID
	}
	return newState, errors.Wrap(err, "error getting last commit ID")
}

func (e *environmentReconciler) getGitRepoCredentials(
	ctx context.Context,
	repoURL string,
) (git.RepoCredentials, error) {
	const repoTypeGit = "git"

	creds := git.RepoCredentials{}

	// NB: This next call returns an empty Repository if no such Repository is
	// found, so instead of continuing to look for credentials if no Repository is
	// found, what we'll do is continue looking for credentials if the Repository
	// we get back doesn't have anything we can use, i.e. no SSH private key or
	// password.
	repo, err := e.argoDB.GetRepository(ctx, repoURL)
	if err != nil {
		return creds, errors.Wrapf(
			err,
			"error getting Repository (Secret) for repo %q",
			repoURL,
		)
	}
	if repo.Type == repoTypeGit || repo.Type == "" {
		creds.SSHPrivateKey = repo.SSHPrivateKey
		creds.Username = repo.Username
		creds.Password = repo.Password
	}
	if creds.SSHPrivateKey == "" && creds.Password == "" {
		// We didn't find any creds yet, so keep looking
		var repoCreds *argocd.RepoCreds
		repoCreds, err = e.argoDB.GetRepositoryCredentials(ctx, repoURL)
		if err != nil {
			return creds, errors.Wrapf(
				err,
				"error getting Repository Credentials (Secret) for repo %q",
				repoURL,
			)
		}
		if repoCreds.Type == repoTypeGit || repoCreds.Type == "" {
			creds.SSHPrivateKey = repoCreds.SSHPrivateKey
			creds.Username = repoCreds.Username
			creds.Password = repoCreds.Password
		}
	}

	// We didn't find any creds, so we're done. We need creds.
	if creds.SSHPrivateKey == "" && creds.Password == "" {
		return creds, errors.Errorf(
			"could not find any credentials for repo %q",
			repoURL,
		)
	}

	return creds, nil
}

func (e *environmentReconciler) getLatestCommit(
	ctx context.Context,
	env *api.Environment,
) (*api.GitCommit, error) {
	if env.Spec.Subscriptions == nil ||
		env.Spec.Subscriptions.Repos == nil ||
		!env.Spec.Subscriptions.Repos.Git {
		return nil, nil
	}

	if env.Spec.GitRepo == nil || env.Spec.GitRepo.URL == "" {
		return nil, errors.New(
			"environment subscribes to a git repo, but does not specify its details",
		)
	}

	repoURL := env.Spec.GitRepo.URL

	creds, err := e.getGitRepoCredentialsFn(ctx, repoURL)
	if err != nil {
		return nil, errors.Wrapf(
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
		return nil, errors.Wrapf(err, "error cloning git repo %q", repoURL)
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
			return nil, errors.Wrapf(
				err,
				"error checking out branch %q from git repo",
				repoURL,
			)
		}
	}
	logger = logger.WithField("branch", branch)
	logger.Debug("checked out branch")

	commit, err := e.getLastCommitIDFn(repo)
	if err != nil {
		if branch != "" {
			return nil, errors.Wrapf(
				err,
				"error determining last commit ID from branch %q of git repo %q",
				branch,
				repoURL,
			)
		}
		return nil, errors.Wrapf(
			err,
			"error determining last commit ID from default branch of git repo %q",
			repoURL,
		)
	}
	logger.WithField("commit", commit).Debug("found latest commit")

	return &api.GitCommit{
		RepoURL: repoURL,
		ID:      commit,
	}, nil
}

func checkoutBranch(repo git.Repo, branch string) error {
	return repo.Checkout(branch)
}

func getLastCommitID(repo git.Repo) (string, error) {
	return repo.LastCommitID()
}
