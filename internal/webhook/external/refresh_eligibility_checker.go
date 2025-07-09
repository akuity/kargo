package external

import (
	"context"
	"regexp"
	"slices"

	"github.com/Masterminds/semver/v3"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	libSemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/internal/controller/warehouses"
	"github.com/akuity/kargo/internal/logging"
	"github.com/expr-lang/expr"
)

// refreshEligibilityChecker encompasses information that came from the inbound 
// request. The checker compares this information against the constraints defined
// in various repo subscription types to determine if a refresh is needed.
type refreshEligibilityChecker struct {
	Git *struct {
		Tag    *git.TagMetadata
		Commit *git.CommitMetadata
		Branch string
		Diffs  []string
	}
	// TODO(Faris): which data type to use
	// GC *kargoapi.GitCommit
	Image *kargoapi.Image
	Chart *kargoapi.Chart
}

// needsRefresh filters out all subscriptions that do not match any of the
// provided repository URLs, and then deletes any subscriptions whos constraints
// are not satisfied by the inbound request data.
func (rc *refreshEligibilityChecker) needsRefresh(
	ctx context.Context,
	subs []kargoapi.RepoSubscription,
	repoURLs ...string,
) bool {
	subs = filterSubsByRepoURL(subs, repoURLs...) // only interested in subs that contain any of the repo URLs.
	return slices.ContainsFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		return rc.matches(ctx, sub)
	})
}

func (rc *refreshEligibilityChecker) matches(
	ctx context.Context,
	sub kargoapi.RepoSubscription,
) bool {
	return rc.matchesGitConstraint(ctx, sub.Git) ||
		rc.matchesImageConstraint(ctx, sub.Image) ||
		rc.matchesChartConstraint(ctx, sub.Chart)
}

// filterSubsByRepoURL deletes all subscriptions from subs that do not
// match any of the provided repository URLs; omitting them from processing.
func filterSubsByRepoURL(
	subs []kargoapi.RepoSubscription,
	repoURLs ...string,
) []kargoapi.RepoSubscription {
	containsRepoURL := func(sub kargoapi.RepoSubscription) bool {
		return sub.Image != nil && slices.Contains(repoURLs, sub.Image.RepoURL) ||
			sub.Git != nil && slices.Contains(repoURLs, sub.Git.RepoURL) ||
			sub.Chart != nil && slices.Contains(repoURLs, sub.Chart.RepoURL)
	}
	return slices.DeleteFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		return !containsRepoURL(sub)
	})
}

func (rc *refreshEligibilityChecker) matchesGitConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	if rc.Git == nil || sub == nil {
		return false
	}
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer:
		return rc.matchesSemVerConstraint(ctx, rc.Git.Tag.Tag, sub.SemverConstraint, sub.StrictSemvers) &&
			rc.matchesGitBaseFilters(ctx, sub)
	case kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategyLexical:
		// We are always dealing with the newest tag in this context (webhooks),
		// so we only need to check if the tag matches the base filters.
		return rc.matchesGitBaseFilters(ctx, sub)
	default: // NewestFromBranch is the default case for Git subscriptions.
		// We are always dealing with the newest commit from the branch in this context
		// (webhooks), so we only need to check if the branch matches the one we are looking for.
		return rc.Git.Branch == sub.Branch && rc.matchesGitBaseFilters(ctx, sub)
	}
}

func (rc *refreshEligibilityChecker) matchesImageConstraint(
	ctx context.Context,
	sub *kargoapi.ImageSubscription,
) bool {
	if rc.Image == nil || sub == nil {
		return false
	}

	switch sub.ImageSelectionStrategy {
	case kargoapi.ImageSelectionStrategyLexical:
		return rc.matchesAllowIgnoreRules(ctx, rc.Image.Tag, sub.AllowTags, sub.IgnoreTags)
	case kargoapi.ImageSelectionStrategyNewestBuild:
		// this strategy is always true in the context of webhooks, as we are
		// always dealing with the newest build of the image.
		return true
	case kargoapi.ImageSelectionStrategyDigest:
		// Unintuitively, the mutable tag name is specified using the semverConstraint field. 
		return rc.Image.Tag == sub.SemverConstraint
	default: // SemVer is the default case for Image subscriptions.
		return rc.matchesSemVerConstraint(ctx, rc.Image.Tag, sub.SemverConstraint, sub.StrictSemvers)
	}
}

func (rc *refreshEligibilityChecker) matchesChartConstraint(
	ctx context.Context,
	sub *kargoapi.ChartSubscription,
) bool {
	if rc.Chart == nil || sub == nil {
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
	return rc.matchesSemVerConstraint(ctx, rc.Image.Tag, sub.SemverConstraint, strict)
}

func (rc *refreshEligibilityChecker) matchesSemVerConstraint(
	ctx context.Context,
	tag string,
	rule string,
	strict bool,
) bool {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"tag", tag,
		"constraint", rule,
		"strictSemvers", strict,
	)

	constraint, err := semver.NewConstraint(rule)
	if err != nil {
		logger.Debug("failed to parse semver constraint", "error", err.Error())
		return false
	}

	version := libSemver.Parse(tag, strict)
	if version == nil {
		logger.Debug("tag is not semver formatted")
		return false
	}
	return constraint.Check(version)
}

// matchesGitBaseFilters checks that path, expression, and tag filters match.
// If there are no path, expression, or tag filters the check returns true.
func (rc *refreshEligibilityChecker) matchesGitBaseFilters(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	return rc.matchesPathFilters(ctx, sub) &&
		rc.matchesAllowIgnoreRules(ctx, rc.Git.Tag.Tag, sub.AllowTags, sub.IgnoreTags) &&
		rc.matchesExpressionFilter(ctx, sub)

}

// matchesPathFilters checks if the provided diffPaths match the
// include and exclude path filters defined in the subscription.
// If there are no include or exclude paths, it returns true.
func (rc *refreshEligibilityChecker) matchesPathFilters(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	if sub.IncludePaths == nil && sub.ExcludePaths == nil {
		return true
	}

	logger := logging.LoggerFromContext(ctx)

	includeSelectors, err := warehouses.GetPathSelectors(sub.IncludePaths)
	if err != nil {
		logger.Error(err, "error parsing include selector",
			"include-selectors", sub.IncludePaths,
		)
		return false
	}

	excludeSelectors, err := warehouses.GetPathSelectors(sub.ExcludePaths)
	if err != nil {
		logger.Error(err, "error parsing exclude selector",
			"exclude-selectors", sub.ExcludePaths,
		)
		return false
	}

	logger.Debug("checking path filters",
		"includeSelectors", includeSelectors,
		"excludeSelectors", excludeSelectors,
	)

	return warehouses.MatchesPathsFilters(
		includeSelectors,
		excludeSelectors,
		rc.Git.Diffs,
	)
}

// matchesExpressionFilter returns true if expression is empty.
// If the expression is not valid an error is logged and false is returned.
// if the tag is not nil, it evaluates the tag metadata against the expression.
// If the commit is not nil, it evaluates the commit metadata against the
// expression.
func (rc *refreshEligibilityChecker) matchesExpressionFilter(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	var matches bool
	switch {
	case sub.ExpressionFilter == "":
		matches = true
	case rc.Git.Tag == nil && rc.Git.Commit == nil:
		matches = true
	default:
		logger := logging.LoggerFromContext(ctx).WithValues(
			"expression", sub.ExpressionFilter,
			"value", sub.ExpressionFilter,
		)

		program, err := expr.Compile(sub.ExpressionFilter)
		if err != nil {
			logger.Error(err, "error compiling tag expression filter")
			return false
		}

		if rc.Git.Tag != nil {
			ok, err := warehouses.EvaluateTagExpression(*rc.Git.Tag, program)
			if err != nil {
				logger.Error(err, "error evaluating tag expression filter")
				return false
			}
			matches = ok
		}

		if rc.Git.Commit != nil {
			ok, err := warehouses.EvaluateCommitExpression(*rc.Git.Commit, program)
			if err != nil {
				logger.Error(err, "error evaluating commit expression filter")
				return false
			}
			matches = ok
		}
	}
	return matches
}

// matchesAllowIgnoreRules checks if the tag matches the allow and ignore rules
// if no allow tags are specified, it returns true.
func (rc *refreshEligibilityChecker) matchesAllowIgnoreRules(
	ctx context.Context,
	tag string,
	allowTags string,
	ignoreTags []string,
) bool {
	logger := logging.LoggerFromContext(ctx)
	if allowTags == "" {
		return true // no allow tags specified, so all tags are allowed
	}

	allowRegex, err := regexp.Compile(allowTags)
	if err != nil {
		logger.Debug("failed to compile allow regex",
			"tag", tag,
			"allow", allowTags,
			"error", err.Error(),
		)
		return false
	}
	return warehouses.Ignores(tag, ignoreTags) || !warehouses.Allows(tag, allowRegex)
}
