package subscription

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/controller/git/commit"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/validation"
)

func init() {
	DefaultSubscriberRegistry.MustRegister(SubscriberRegistration{
		Predicate: func(
			_ context.Context,
			sub kargoapi.RepoSubscription,
		) (bool, error) {
			return sub.Git != nil, nil
		},
		Value:    newGitSubscriber,
		Metadata: SubscriberMetadata{Schema: mustLoadSchema("git")},
	})
}

// gitSubscriber is an implementation of the Subscriber interface that discovers
// commits from a Git repository.
type gitSubscriber struct {
	credentialsDB credentials.Database

	// newSelectorFn constructs the commit Selector for a subscription. It is a
	// field so tests can substitute a fake Selector for the real one.
	newSelectorFn func(
		ctx context.Context,
		sub kargoapi.GitSubscription,
		creds *git.RepoCredentials,
	) (commit.Selector, error)
}

// newGitSubscriber returns an implementation of the Subscriber interface that
// discovers commits from a Git repository.
func newGitSubscriber(
	_ context.Context,
	credentialsDB credentials.Database,
) (Subscriber, error) {
	return &gitSubscriber{
		credentialsDB: credentialsDB,
		newSelectorFn: commit.NewSelector,
	}, nil
}

// ApplySubscriptionDefaults implements Subscriber.
func (g *gitSubscriber) ApplySubscriptionDefaults(
	_ context.Context,
	sub *kargoapi.RepoSubscription,
) error {
	if sub == nil || sub.Git == nil {
		return nil
	}
	if sub.Git.CommitSelectionStrategy == "" {
		sub.Git.CommitSelectionStrategy = kargoapi.CommitSelectionStrategyNewestFromBranch
	}
	if sub.Git.StrictSemvers == nil {
		sub.Git.StrictSemvers = ptr.To(true)
	}
	if sub.Git.DiscoveryLimit == 0 {
		sub.Git.DiscoveryLimit = 20
	}
	return nil
}

// ValidateSubscription implements Subscriber. The subscription's schema
// covers everything else.
func (g *gitSubscriber) ValidateSubscription(
	_ context.Context,
	f *field.Path,
	s kargoapi.RepoSubscription,
) field.ErrorList {
	if err := validation.SemverConstraint(
		f.Child("semverConstraint"),
		s.Git.SemverConstraint,
	); err != nil {
		return field.ErrorList{err}
	}
	return nil
}

// DiscoverArtifacts implements Subscriber.
func (g *gitSubscriber) DiscoverArtifacts(
	ctx context.Context,
	project string,
	sub kargoapi.RepoSubscription,
	last any,
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
			Username: creds.Username,
			Password: creds.Password,
		}
		logger.Debug("obtained credentials for git repo")
	} else {
		logger.Debug("found no credentials for git repo")
	}

	selector, err := g.newSelectorFn(ctx, *gitSub, repoCreds)
	if err != nil {
		return nil, fmt.Errorf(
			"error obtaining selector for commits from git repo %q: %w",
			gitSub.RepoURL, err,
		)
	}

	// Observe the current remote ref state with a single ls-remote round-trip,
	// before committing to an expensive clone. If it matches what we recorded at
	// the previous successful discovery, nothing relevant has moved and the
	// previously selected commits remain valid -- so we can skip the clone.
	observedRefs, err := selector.ListRefs(ctx)
	if err != nil {
		// Treat ls-remote failure as non-fatal: fall through to the clone path,
		// which preserves today's behavior (and will surface any real
		// connectivity error there).
		logger.Debug("error listing remote refs; proceeding to clone", "err", err)
	} else if prev, ok := last.(kargoapi.GitDiscoveryResult); ok &&
		prev.RepoURL == gitSub.RepoURL && observedRefs != nil &&
		gitRefsEqual(prev.ObservedRefs, observedRefs) {
		// TODO(krancour): The prev.RepoURL == gitSub.RepoURL above safeguards
		// against the fragility of the caller's choice to correlate prior discovery
		// results with subscriptions positionally. That should be revisited
		// regardless, but is deferred until Warehouses are permitted multiple
		// subscriptions to the same repository as that will require further changes
		// to the same mechanism.
		//
		// See https://github.com/akuity/kargo/issues/6724.
		logger.Debug(
			"remote refs unchanged since last discovery; skipping clone",
			"commits", len(prev.Commits),
		)
		return kargoapi.GitDiscoveryResult{
			RepoURL:          gitSub.RepoURL,
			Commits:          prev.Commits,
			SubscriptionName: sub.Name,
			ObservedRefs:     observedRefs,
		}, nil
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
		RepoURL:          gitSub.RepoURL,
		Commits:          commits,
		SubscriptionName: sub.Name,
		ObservedRefs:     observedRefs,
	}, nil
}

// gitRefsEqual reports whether two recorded ref observations are equal. A nil
// observation (e.g. recorded before this feature existed, or suppressed because
// a subscription's tag set exceeded the cap) never compares equal, so discovery
// always falls through to a clone in those cases.
func gitRefsEqual(a, b *kargoapi.GitDiscoveryRefs) bool {
	if a == nil || b == nil {
		return false
	}
	if a.BranchHead != b.BranchHead || len(a.Tags) != len(b.Tags) {
		return false
	}
	for i := range a.Tags {
		if a.Tags[i] != b.Tags[i] {
			return false
		}
	}
	return true
}
