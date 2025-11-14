package warehouses

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/controller/git/commit"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

// discoverCommits discovers commits from the given Git repositories based on the
// given subscriptions. It returns a list of GitDiscoveryResult objects, each
// containing the discovered commits for the corresponding subscription.
func (r *reconciler) discoverCommits(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.GitDiscoveryResult, error) {
	results := make([]kargoapi.GitDiscoveryResult, 0, len(subs))

	for _, s := range subs {
		if s.Git == nil {
			continue
		}

		sub := *s.Git

		logger := logging.LoggerFromContext(ctx).WithValues("repo", sub.RepoURL)

		// Obtain credentials for the Git repository.
		creds, err := r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, sub.RepoURL)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining credentials for git repo %q: %w",
				sub.RepoURL,
				err,
			)
		}
		var repoCreds *git.RepoCredentials
		if creds != nil {
			repoCreds = &git.RepoCredentials{
				Username:      creds.Username,
				Password:      creds.Password,
				SSHPrivateKey: creds.SSHPrivateKey,
			}
			logger.Debug("obtained credentials for git repo")
		} else {
			logger.Debug("found no credentials for git repo")
		}

		selector, err := commit.NewSelector(ctx, *s.Git, repoCreds)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining selector for commits from git repo %q: %w",
				sub.RepoURL, err,
			)
		}
		commits, err := selector.Select(ctx)
		if err != nil {
			return nil, fmt.Errorf(
				"error discovering commits from git repo %q: %w",
				sub.RepoURL, err,
			)
		}

		if len(commits) == 0 {
			results = append(results, kargoapi.GitDiscoveryResult{
				RepoURL: sub.RepoURL,
			})
			logger.Debug("discovered no commits")
			continue
		}

		results = append(results, kargoapi.GitDiscoveryResult{
			RepoURL: sub.RepoURL,
			Commits: commits,
		})
		logger.Debug(
			"discovered commits",
			"count", len(commits),
		)
	}

	return results, nil
}
