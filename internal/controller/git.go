package controller

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/credentials"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/logging"
)

func (e *environmentReconciler) applyGitRepoUpdate(
	ctx context.Context,
	namespace string,
	newState api.EnvironmentState,
	update api.GitRepoUpdate,
) (api.EnvironmentState, error) {
	logger := logging.LoggerFromContext(ctx).WithField("repo", update.RepoURL)

	creds, ok, err :=
		e.credentialsDB.Get(ctx, namespace, credentials.TypeGit, update.RepoURL)
	if err != nil {
		return newState, errors.Wrapf(
			err,
			"error obtaining credentials for git repo %q",
			update.RepoURL,
		)
	}
	var repoCreds *git.Credentials
	if ok {
		repoCreds = &git.Credentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
		logger.Debug("obtained credentials for git repo")
	} else {
		logger.Debug("found no credentials for git repo")
	}

	commitID, err := e.gitApplyUpdateFn(update.RepoURL, update.Branch, repoCreds,
		func(homeDir, workingDir string) (string, error) {
			if update.Kustomize != nil {
				if err = e.applyKustomize(
					newState,
					*update.Kustomize,
					workingDir,
				); err != nil {
					if update.Branch == "" {
						return "", errors.Wrapf(
							err,
							"error updating git repository %q via Kustomize",
							update.RepoURL,
						)
					}
					return "", errors.Wrapf(
						err,
						"error updating branch %q in git repository %q via Kustomize",
						update.Branch,
						update.RepoURL,
					)
				}
			}

			if update.Helm != nil {
				if err = e.applyHelm(
					newState,
					*update.Helm,
					homeDir,
					workingDir,
				); err != nil {
					if update.Branch == "" {
						return "", errors.Wrapf(
							err,
							"error updating git repository %q via Helm",
							update.RepoURL,
						)
					}
					return "", errors.Wrapf(
						err,
						"error updating branch %q in git repository %q via Helm",
						update.Branch,
						update.RepoURL,
					)
				}
			}

			// TODO: This is an awful commit message! Fix it!
			return "kargo made some changes!", nil
		},
	)
	if err != nil {
		return newState, err
	}

	// Only try to update state if commitID isn't empty. If it's empty, it
	// indicates no change was committed to the repository and there's nothing to
	// update here.
	if commitID != "" {
		logger.WithField("commit", commitID).Debug("pushed new commit to repo")
		for i := range newState.Commits {
			if newState.Commits[i].RepoURL == update.RepoURL {
				newState.Commits[i].ID = commitID
			}
		}
	} else {
		logger.Debug("no changes pushed to repo")
	}

	return newState, nil
}

func (e *environmentReconciler) getLatestCommits(
	ctx context.Context,
	namespace string,
	subs []api.GitSubscription,
) ([]api.GitCommit, error) {
	latestCommits := make([]api.GitCommit, len(subs))
	for i, sub := range subs {
		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)
		creds, ok, err :=
			e.credentialsDB.Get(ctx, namespace, credentials.TypeGit, sub.RepoURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for git repo %q",
				sub.RepoURL,
			)
		}
		var repoCreds *git.Credentials
		if ok {
			repoCreds = &git.Credentials{
				Username:      creds.Username,
				Password:      creds.Password,
				SSHPrivateKey: creds.SSHPrivateKey,
			}
			logger.Debug("obtained credentials for git repo")
		} else {
			logger.Debug("found no credentials for git repo")
		}

		commit, err := e.getLatestCommitIDFn(sub.RepoURL, sub.Branch, repoCreds)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error determining latest commit ID of git repo %q",
				sub.RepoURL,
			)
		}
		logger.WithField("commit", commit).
			Debug("found latest commit from repo")
		latestCommits[i] = api.GitCommit{
			RepoURL: sub.RepoURL,
			ID:      commit,
		}
	}
	return latestCommits, nil
}
