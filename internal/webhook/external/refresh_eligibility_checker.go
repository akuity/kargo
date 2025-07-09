package external

import (
	"context"
	"regexp"
	"slices"

	"github.com/Masterminds/semver/v3"
	"github.com/expr-lang/expr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libGit "github.com/akuity/kargo/internal/controller/git"
	libSemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/internal/controller/warehouses"
	"github.com/akuity/kargo/internal/logging"
)

// refreshEligibilityChecker encompasses information that came from the inbound
// request. The checker compares this information against the constraints defined
// in various repo subscription types to determine if a refresh is needed.
type refreshEligibilityChecker struct {
	git   *codeChange
	image *imageChange
	chart *chartChange
}

func (rc *refreshEligibilityChecker) isValid(changeType any) bool {
	switch changeType := changeType.(type) {
	case *codeChange:
		return rc.git != nil && rc.git.tag != nil && rc.git.tag.Tag != "" &&
			rc.git.commit == changeType.commit && rc.git.branch == changeType.branch &&
			slices.Equal(rc.git.diffs, changeType.diffs)
	case *imageChange:
		return rc.image != nil && rc.image.tag == changeType.tag &&
			rc.image.digest == changeType.digest
	case *chartChange:
		return rc.chart != nil && rc.chart.tag == changeType.tag
	default:
		return false
	}
}

type codeChange struct {
	tag    *libGit.TagMetadata
	commit *libGit.CommitMetadata
	branch string
	diffs  []string
}

type imageChange struct {
	tag    string
	digest string
}

type chartChange struct {
	tag string
}

// needsRefresh filters out all subscriptions that do not match any of the
// provided repository URLs and then evaluates whether any of the remaining
// subscriptions' constraints are satisfied by the inbound request data.
// If any of the subscriptions match, it returns true, indicating that a refresh
// is needed. If no subscriptions match, it returns false.
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
			sub.Chart != nil && slices.Contains(repoURLs, sub.Chart.RepoURL)
	}
	return slices.DeleteFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		return !containsRepoURL(sub)
	})
}

func (rc *refreshEligibilityChecker) matchesGitConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	if rc.git == nil || sub == nil {
		return false
	}
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer:
		return rc.matchesSemVerConstraint(ctx, rc.git.tag.Tag, sub.SemverConstraint, sub.StrictSemvers) &&
			rc.matchesGitBaseFilters(ctx, sub)
	case kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategyLexical:
		// We are always dealing with the newest tag in this context (webhooks),
		// so we only need to check if the tag matches the base filters.
		return rc.matchesGitBaseFilters(ctx, sub)
	default: // NewestFromBranch is the default case for Git subscriptions.
		// We are always dealing with the newest commit from the branch in this context
		// (webhooks), so we only need to check if the branch matches the one we are looking for.
		logger := logging.LoggerFromContext(ctx).WithValues(
			"branch", rc.git.branch,
			"target-branch", sub.Branch,
		)
		if rc.git.branch != sub.Branch {
			logger.Info("branch does not match subscription branch")
			return false
		}
		logger.Info("branch matches subscription branch")
		return rc.matchesGitBaseFilters(ctx, sub)
	}
}

func (rc *refreshEligibilityChecker) matchesImageConstraint(ctx context.Context, sub *kargoapi.ImageSubscription) bool {
	if rc.image == nil || sub == nil {
		return false
	}

	switch sub.ImageSelectionStrategy {
	case kargoapi.ImageSelectionStrategyLexical:
		return rc.matchesAllowIgnoreRules(ctx, rc.image.tag, sub.AllowTags, sub.IgnoreTags)
	case kargoapi.ImageSelectionStrategyNewestBuild:
		// this strategy is always true in the context of webhooks, as we are
		// always dealing with the newest build of the image.
		return true
	case kargoapi.ImageSelectionStrategyDigest:
		// Unintuitively, the mutable tag name is specified using the semverConstraint field.
		return rc.image.tag == sub.SemverConstraint
	default: // SemVer is the default case for Image subscriptions.
		return rc.matchesSemVerConstraint(ctx, rc.image.tag, sub.SemverConstraint, sub.StrictSemvers)
	}
}

func (rc *refreshEligibilityChecker) matchesChartConstraint(ctx context.Context, sub *kargoapi.ChartSubscription) bool {
	if rc.chart == nil || sub == nil {
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
	return rc.matchesSemVerConstraint(ctx, rc.chart.tag, sub.SemverConstraint, strict)
}

func (rc *refreshEligibilityChecker) matchesSemVerConstraint(ctx context.Context, tag, rule string, strict bool) bool {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"tag", tag,
		"constraint", rule,
		"strictSemvers", strict,
	)

	constraint, err := semver.NewConstraint(rule)
	if err != nil {
		logger.Error(err, "failed to parse semver constraint")
		return false
	}

	version := libSemver.Parse(tag, strict)
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

// matchesGitBaseFilters checks that path, expression, and tag filters match.
func (rc *refreshEligibilityChecker) matchesGitBaseFilters(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	logger := logging.LoggerFromContext(ctx)
	if ok := rc.matchesPathFilters(ctx, sub); !ok {
		logger.Debug("path filters not satisfied")
		return false
	}
	logger.Debug("path filters satisfied")

	if ok := rc.matchesAllowIgnoreRules(ctx, rc.git.tag.Tag, sub.AllowTags, sub.IgnoreTags); !ok {
		logger.Debug("allow/ignore rules not satisfied")
		return false
	}
	logger.Debug("allow/ignore rules satisfied")

	if ok := rc.matchesExpressionFilter(ctx, sub); !ok {
		logger.Debug("expression filters not satisfied")
		return false
	}
	logger.Debug("expression filters satisfied")
	return true

}

// matchesPathFilters checks if the provided diffPaths match the
// include and exclude path filters defined in the subscription.
// If there are no include or exclude paths, it returns true.
func (rc *refreshEligibilityChecker) matchesPathFilters(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	logger := logging.LoggerFromContext(ctx)

	if sub.IncludePaths == nil && sub.ExcludePaths == nil {
		logger.Debug("no path filters specified, all paths are allowed")
		return true
	}

	logger.WithValues(
		"includePaths", sub.IncludePaths,
		"excludePaths", sub.ExcludePaths,
	)

	includeSelectors, err := warehouses.GetPathSelectors(sub.IncludePaths)
	if err != nil {
		logger.Error(err, "error parsing include selector")
		return false
	}

	excludeSelectors, err := warehouses.GetPathSelectors(sub.ExcludePaths)
	if err != nil {
		logger.Error(err, "error parsing exclude selector")
		return false
	}

	return warehouses.MatchesPathsFilters(
		includeSelectors,
		excludeSelectors,
		rc.git.diffs,
	)
}

// matchesExpressionFilter returns true if expression is empty.
// If the expression is not valid an error is logged and false is returned.
// if the tag is not nil, it evaluates the tag metadata against the expression.
// If the commit is not nil, it evaluates the commit metadata against the
// expression.
func (rc *refreshEligibilityChecker) matchesExpressionFilter(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"expression", sub.ExpressionFilter,
		"value", sub.ExpressionFilter,
	)

	if sub.ExpressionFilter == "" {
		return true // no expression filter specified, so all tags are allowed
	}

	program, err := expr.Compile(sub.ExpressionFilter)
	if err != nil {
		logger.Error(err, "error compiling tag expression filter")
		return false
	}

	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer,
		kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategyLexical:
		matches, err := warehouses.EvaluateTagExpression(*rc.git.tag, program)
		if err != nil {
			logger.Error(err, "error evaluating tag expression filter")
			return false
		}
		return matches
	default:
		matches, err := warehouses.EvaluateCommitExpression(*rc.git.commit, program)
		if err != nil {
			logger.Error(err, "error evaluating commit expression filter")
			return false
		}
		return matches
	}
}

// matchesAllowIgnoreRules checks if the tag matches the allow and ignore rules
// if no allow tags are specified, it returns true.
func (rc *refreshEligibilityChecker) matchesAllowIgnoreRules(
	ctx context.Context,
	tag string,
	allowTags string,
	ignoreTags []string,
) bool {
	logger := logging.LoggerFromContext(ctx).WithValues(
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

	allowed := warehouses.Allows(tag, allowRegex)
	if !allowed {
		logger.Debug("tag found on allow list")
		return false
	}
	return true
}
