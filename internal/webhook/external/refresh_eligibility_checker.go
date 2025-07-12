package external

import (
	"context"
	"regexp"
	"slices"

	"github.com/Masterminds/semver/v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libSemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/internal/controller/warehouses"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/logging"
)

// refreshEligibilityChecker encompasses information that came from the inbound
// request. The checker compares this information against the constraints defined
// in various repo subscription types to determine if a refresh is needed.
type refreshEligibilityChecker struct {
	branchName  *string
	newGitTag   *string
	newImageTag *string
	newChartTag *string
}

// needsRefresh filters out all subscriptions that do not match any of the
// provided repository URLs and then evaluates whether any of the remaining
// subscriptions' constraints are satisfied by the inbound request data;
// indicating whether a refresh is needed.
func (rc *refreshEligibilityChecker) needsRefresh(
	ctx context.Context,
	subs []kargoapi.RepoSubscription,
	repoURLs ...string,
) bool {
	subs = filterSubsByRepoURL(subs, repoURLs...) // only interested in subs that contain any of the repo URLs.
	return slices.ContainsFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		var shouldRefresh bool
		switch {
		case sub.Git != nil:
			shouldRefresh = rc.matchesGitConstraint(ctx, sub.Git)
		case sub.Image != nil:
			shouldRefresh = rc.matchesImageConstraint(ctx, sub.Image)
		case sub.Chart != nil:
			shouldRefresh = rc.matchesChartConstraint(ctx, sub.Chart)
		}
		return shouldRefresh
	})
}

// filterSubsByRepoURL deletes all subscriptions from subs that do not
// match any of the provided repository URLs; omitting them from processing.
func filterSubsByRepoURL(subs []kargoapi.RepoSubscription, repoURLs ...string) []kargoapi.RepoSubscription {
	containsRepoURL := func(sub kargoapi.RepoSubscription) bool {
		return sub.Image != nil && slices.Contains(repoURLs, sub.Image.RepoURL) ||
			sub.Git != nil && slices.Contains(repoURLs, sub.Git.RepoURL) ||
			sub.Chart != nil && slices.Contains(repoURLs,
				helm.NormalizeChartRepositoryURL(sub.Chart.RepoURL),
			)
	}
	return slices.DeleteFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		return !containsRepoURL(sub)
	})
}

func (rc *refreshEligibilityChecker) matchesGitConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	if sub == nil {
		return false
	}
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer:
		return rc.matchesSemVerConstraint(ctx, rc.newGitTag, sub.SemverConstraint, sub.StrictSemvers) &&
			rc.matchesAllowIgnoreRules(ctx, rc.newGitTag, sub.AllowTags, sub.IgnoreTags)
	case kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategyLexical:
		return rc.matchesAllowIgnoreRules(ctx, rc.newGitTag, sub.AllowTags, sub.IgnoreTags)
	default: // NewestFromBranch is the default case for Git subscriptions.
		return rc.matchesNewestBranchConstraint(ctx, sub) &&
			rc.matchesAllowIgnoreRules(ctx, rc.newGitTag, sub.AllowTags, sub.IgnoreTags)
	}
}

func (rc *refreshEligibilityChecker) matchesImageConstraint(ctx context.Context, sub *kargoapi.ImageSubscription) bool {
	if sub == nil {
		return false
	}

	switch sub.ImageSelectionStrategy {
	case kargoapi.ImageSelectionStrategyLexical:
		return rc.matchesAllowIgnoreRules(ctx, rc.newImageTag, sub.AllowTags, sub.IgnoreTags)
	case kargoapi.ImageSelectionStrategyNewestBuild:
		// this strategy is always true in the context of webhooks, as we are
		// always dealing with the newest build of the image.
		return true
	case kargoapi.ImageSelectionStrategyDigest:
		// Unintuitively, the mutable tag name is specified using the semverConstraint field.
		return rc.newImageTag == nil || *rc.newImageTag == sub.SemverConstraint
	default: // SemVer is the default case for Image subscriptions.
		return rc.matchesSemVerConstraint(ctx, rc.newImageTag, sub.SemverConstraint, sub.StrictSemvers)
	}
}

func (rc *refreshEligibilityChecker) matchesChartConstraint(ctx context.Context, sub *kargoapi.ChartSubscription) bool {
	if rc.newChartTag == nil || sub == nil {
		return false
	}
	// " If left unspecified, the subscription implicitly selects the semantically greatest version of the chart."
	//		source: https://docs.kargo.io/user-guide/how-to-guides/working-with-warehouses
	//
	// Since we're always working with the greatest version of the chart in the context (webhooks),
	// we can simply return true if the semverConstraint is empty.
	if sub.SemverConstraint == "" {
		return true
	}
	strict := true // SemVer constraints are always strict for charts.
	return rc.matchesSemVerConstraint(ctx, rc.newChartTag, sub.SemverConstraint, strict)
}

func (rc *refreshEligibilityChecker) matchesSemVerConstraint(
	ctx context.Context,
	tag *string,
	rule string,
	strict bool,
) bool {
	logger := logging.LoggerFromContext(ctx)
	if tag == nil {
		logger.Debug("tag is nil, skipping semver constraint check")
		return true // no tag to match against
	}

	logger = logger.WithValues(
		"tag", tag,
		"constraint", rule,
		"strictSemvers", strict,
	)

	if rule == "" {
		logger.Debug("semver constraint rule is unset, skipping semver constraint check rule check")
		return true // no semver constraint specified, so all tags are allowed
	}

	constraint, err := semver.NewConstraint(rule)
	if err != nil {
		logger.Error(err, "failed to parse semver constraint")
		return false
	}

	version := libSemver.Parse(*tag, strict)
	if version == nil {
		logger.Info("tag is not semver formatted")
		return false
	}

	matches := constraint.Check(version)
	if !matches {
		logger.Info("tag does not match semver constraint")
	}
	return matches
}

func (rc *refreshEligibilityChecker) matchesNewestBranchConstraint(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
) bool {
	logger := logging.LoggerFromContext(ctx)
	if rc.branchName == nil {
		logger.Debug("branch name is nil, skipping branch constraint check")
		return true
	}

	logger = logger.WithValues(
		"branch", *rc.branchName,
		"target-branch", sub.Branch,
	)

	if *rc.branchName != sub.Branch {
		logger.Debug("branch does not match subscription branch")
		return false
	}
	logger.Debug("branch matches subscription branch")
	return true
}

// matchesAllowIgnoreRules checks if the tag matches the allow and ignore rules
// if no allow tags are specified, it returns true.
func (rc *refreshEligibilityChecker) matchesAllowIgnoreRules(
	ctx context.Context,
	tag *string,
	allowTags string,
	ignoreTags []string,
) bool {
	logger := logging.LoggerFromContext(ctx)
	if tag == nil {
		logger.Debug("tag is nil, skipping allow/ignore rules check")
		return true // no tag to match against
	}

	logger = logger.WithValues(
		"tag", tag,
		"allowTags", allowTags,
		"ignoreTags", ignoreTags,
	)

	if allowTags == "" {
		logger.Debug("no allow tags specified, all tags are allowed")
		return true
	}

	allowRegex, err := regexp.Compile(allowTags)
	if err != nil {
		logger.Error(err, "failed to compile allow regex")
		return false
	}

	allowed := warehouses.Allows(*tag, allowRegex)
	if !allowed {
		logger.Debug("tag found on allow list")
		return false
	}
	return true
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
