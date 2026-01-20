package subscription

import (
	"context"
	"fmt"
	"regexp"
	"slices"

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

var (
	// nolint: lll
	gitURLRegex = regexp.MustCompile(`(?:^(ssh|https?)://(?:([\w-]+)(:(.+))?@)?([\w-]+(?:\.[\w-]+)*)(?::(\d{1,5}))?(/.*)$)|(?:^([\w-]+)@([\w+]+(?:\.[\w-]+)*):(/?.*))`)
	branchRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._\/-]*[a-zA-Z0-9_-])?$`)

	validCommitSelectionStrategies = []kargoapi.CommitSelectionStrategy{
		kargoapi.CommitSelectionStrategyLexical,
		kargoapi.CommitSelectionStrategyNewestFromBranch,
		kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategySemVer,
	}
)

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

// ValidateSubscription implements Subscriber.
func (g *gitSubscriber) ValidateSubscription(
	_ context.Context,
	f *field.Path,
	s kargoapi.RepoSubscription,
) field.ErrorList {
	// TODO(krancour): Longer term, we might want to start doing this with JSON
	// schema.

	sub := s.Git
	var errs field.ErrorList

	// Validate RepoURL: MinLength=1, Pattern (Git URL regex)
	if err := validation.MinLength(f.Child("repoURL"), sub.RepoURL, 1); err != nil {
		errs = append(errs, err)
	}
	if !gitURLRegex.MatchString(sub.RepoURL) {
		errs = append(errs, field.Invalid(
			f.Child("repoURL"),
			sub.RepoURL,
			"must be a valid Git repository URL",
		))
	}

	// Validate Branch: MinLength=1, MaxLength=255, Pattern
	if sub.Branch != "" {
		if err := validation.MinLength(f.Child("branch"), sub.Branch, 1); err != nil {
			errs = append(errs, err)
		}
		if err := validation.MaxLength(f.Child("branch"), sub.Branch, 255); err != nil {
			errs = append(errs, err)
		}
		if !branchRegex.MatchString(sub.Branch) {
			errs = append(errs, field.Invalid(
				f.Child("branch"),
				sub.Branch,
				"must be a valid branch name",
			))
		}
	}

	// Validate CommitSelectionStrategy is one of the allowed values
	if sub.CommitSelectionStrategy != "" {
		if err := validateCommitSelectionStrategy(
			f.Child("commitSelectionStrategy"),
			sub.CommitSelectionStrategy,
		); err != nil {
			errs = append(errs, err)
		}
	}

	// Validate SemverConstraint
	if err := validation.SemverConstraint(
		f.Child("semverConstraint"),
		sub.SemverConstraint,
	); err != nil {
		errs = append(errs, err)
	}

	// Validate DiscoveryLimit: Minimum=1, Maximum=100
	if sub.DiscoveryLimit < 1 {
		errs = append(errs, field.Invalid(
			f.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be >= 1",
		))
	} else if sub.DiscoveryLimit > 100 {
		errs = append(errs, field.Invalid(
			f.Child("discoveryLimit"),
			sub.DiscoveryLimit,
			"must be <= 100",
		))
	}

	return errs
}

func validateCommitSelectionStrategy(
	f *field.Path,
	strategy kargoapi.CommitSelectionStrategy,
) *field.Error {
	if !slices.Contains(validCommitSelectionStrategies, strategy) {
		return field.NotSupported(f, strategy, []string{
			string(kargoapi.CommitSelectionStrategyLexical),
			string(kargoapi.CommitSelectionStrategyNewestFromBranch),
			string(kargoapi.CommitSelectionStrategyNewestTag),
			string(kargoapi.CommitSelectionStrategySemVer),
		})
	}
	return nil
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
