package subscription

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/controller/git/commit"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

func init() {
	DefaultSubscriberRegistry.MustRegister(SubscriberRegistration{
		Predicate: func(
			_ context.Context,
			sub kargoapi.RepoSubscription,
		) (bool, error) {
			return sub.Git != nil, nil
		},
		Value: newGitSubscriber,
	})
}

// gitSubscriber is an implementation of the Subscriber interface that discovers
// commits from a Git repository.
type gitSubscriber struct {
	credentialsDB credentials.Database
}

// newGitSubscriber returns an implementation of the Subscriber interface that
// discovers commits from a Git repository.
func newGitSubscriber(
	_ context.Context,
	credentialsDB credentials.Database,
) (Subscriber, error) {
	return &gitSubscriber{credentialsDB: credentialsDB}, nil
}

// DiscoverArtifacts implements Subscriber.
func (g *gitSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
) (any, error) {
	gitSub := sub.Git

	if gitSub == nil {
		return nil, nil
	}

	logger := logging.LoggerFromContext(ctx).WithValues("repo", gitSub.RepoURL)

	// Obtain credentials for the Git repository.
	creds, err := g.credentialsDB.Get(
		ctx,
		project,
		credentials.TypeGit,
		gitSub.RepoURL,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining credentials for git repo %q: %w",
			gitSub.RepoURL, err,
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

	selector, err := commit.NewSelector(ctx, *gitSub, repoCreds)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining selector for commits from git repo %q: %w",
			gitSub.RepoURL, err,
		)
	}
	commits, err := selector.Select(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error discovering commits from git repo %q: %w",
			gitSub.RepoURL, err,
		)
	}
	if len(commits) == 0 {
		logger.Debug("discovered no commits")
	} else {
		logger.Debug("discovered commits", "count", len(commits))
	}

	return kargoapi.GitDiscoveryResult{
		RepoURL: gitSub.RepoURL,
		Commits: commits,
	}, nil
}
