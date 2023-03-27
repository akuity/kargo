package environments

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/credentials"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/logging"
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
		}
	}
	return latestCommits, nil
}
