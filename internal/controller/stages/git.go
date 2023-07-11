package stages

import (
	"context"

	"github.com/pkg/errors"

	"github.com/akuity/bookkeeper/pkg/git"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) getLatestCommits(
	ctx context.Context,
	namespace string,
	subs []api.GitSubscription,
) ([]api.GitCommit, error) {
	latestCommits := make([]api.GitCommit, len(subs))
	for i, sub := range subs {
		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)
		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, sub.RepoURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for git repo %q",
				sub.RepoURL,
			)
		}
		var repoCreds *git.RepoCredentials
		if ok {
			repoCreds = &git.RepoCredentials{
				Username:      creds.Username,
				Password:      creds.Password,
				SSHPrivateKey: creds.SSHPrivateKey,
			}
			logger.Debug("obtained credentials for git repo")
		} else {
			logger.Debug("found no credentials for git repo")
		}

		commit, err := r.getLatestCommitIDFn(sub.RepoURL, sub.Branch, repoCreds)
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
			Branch:  sub.Branch,
		}
	}
	return latestCommits, nil
}

func getLatestCommitID(
	repoURL string,
	branch string,
	creds *git.RepoCredentials,
) (string, error) {
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	repo, err := git.Clone(repoURL, *creds)
	if err != nil {
		return "", errors.Wrapf(err, "error cloning git repo %q", repoURL)

	}
	if branch != "" {
		if err = repo.Checkout(branch); err != nil {
			return "", errors.Wrapf(
				err,
				"error checking out branch %q from git repo",
				repoURL,
			)
		}
	}
	commit, err := repo.LastCommitID()
	if branch != "" {
		return commit, errors.Wrapf(
			err,
			"error determining last commit ID from branch %q of git repo %q",
			branch,
			repoURL,
		)
	}
	return commit, errors.Wrapf(
		err,
		"error determining last commit ID from default branch of git repo %q",
		repoURL,
	)
}
