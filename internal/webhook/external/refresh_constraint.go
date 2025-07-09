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

type refreshConstraint struct {
	Git *struct {
		Tag        *git.TagMetadata
		Commit     *git.CommitMetadata
		Branch     string
		Diffs      []string
	}
	Image *struct {
		Tag string
	}
	Chart *struct {
		Tag string
	}
}

func (rc refreshConstraint) needsRefresh(
	ctx context.Context,
	subs []kargoapi.RepoSubscription,
	repoURLs ...string,
) bool {
	subs = filterSubsByRepoURL(subs, repoURLs...) // only interested in subs that contain any of the repo URLs.
	return slices.ContainsFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		return rc.matches(ctx, sub)
	})
}

// filterSubsByRepoURL deletes all subscriptions from subs that do not
// match any of the provided repository URLs.
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

func (rc refreshConstraint) matches(
	ctx context.Context,
	sub kargoapi.RepoSubscription,
) bool {
	return rc.matchesGitConstraint(ctx, sub.Git) ||
		rc.matchesImageConstraint(sub.Image) ||
		rc.matchesChartConstraint(sub.Chart)
}

func (rc refreshConstraint) matchesGitConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	if rc.Git == nil || sub == nil {
		return false
	}
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategyLexical:
		return rc.matchesLexicalConstraint(ctx, sub)
	case kargoapi.CommitSelectionStrategySemVer:
		return rc.matchesSemVerConstraint(ctx, sub)
	case kargoapi.CommitSelectionStrategyNewestTag:
		return rc.matchesNewestTagConstraint(ctx, sub)
	default:
		return rc.matchesNewestFromBranchConstraint(ctx, sub)
	}
}

func (rc refreshConstraint) matchesSemVerConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"tag", rc.Git.Tag.Tag,
		"constraint", sub.SemverConstraint,
		"strictSemvers", sub.StrictSemvers,
	)

	constraint, err := semver.NewConstraint(sub.SemverConstraint)
	if err != nil {
		logger.Debug("failed to parse semver constraint", "error", err.Error())
		return false
	}

	version := libSemver.Parse(rc.Git.Tag.Tag, sub.StrictSemvers)
	if version == nil {
		logger.Debug("tag is not semver formatted")
		return false
	}

	if !constraint.Check(version) {
		logger.Debug("tag does not satisfy semver constraint")
		return false
	}
	return rc.matchesBaseFilters(ctx, sub)
}

func (rc refreshConstraint) matchesLexicalConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	// base filters are enough here because we are only dealing with one tag e.g. no sorting required.
	return rc.matchesBaseFilters(ctx, sub)
}

func (rc refreshConstraint) matchesNewestFromBranchConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	// we are only dealing with the newest commit from the branch, so we only need to check if the branch matches
	// the one we are looking for + the base filters.
	return rc.Git.Branch == sub.Branch && rc.matchesBaseFilters(ctx, sub)
}

func (rc refreshConstraint) matchesNewestTagConstraint(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	// we are always dealing with the newest tag in this context (webhooks),
	// so we only need to check if the tag matches the base filters.
	return rc.matchesBaseFilters(ctx, sub)
}

func (rc refreshConstraint) matchesImageConstraint(
	sub *kargoapi.ImageSubscription,
) bool {
	if rc.Image == nil || sub == nil {
		return false
	}
	// get strategy

	// check for match
	return false
}

func (rc refreshConstraint) matchesChartConstraint(
	sub *kargoapi.ChartSubscription,
) bool {
	if rc.Chart == nil || sub == nil {
		return false
	}
	// get strategy

	// check for match
	return false
}

// matchesBaseFilters checks that path, expression, and tag filters match.
// If there are no path, expression, or tag filters the check returns true.
func (rc refreshConstraint) matchesBaseFilters(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	return rc.matchesPathFilters(ctx, sub) &&
		rc.matchesExpressionFilter(ctx, sub) &&
		rc.matchesAllowIgnoreRules(ctx, sub)
}

// matchesPathFilters checks if the provided diffPaths match the
// include and exclude path filters defined in the subscription.
// If there are no include or exclude paths, it returns true.
func (rc refreshConstraint) matchesPathFilters(ctx context.Context, sub *kargoapi.GitSubscription) bool {
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

// matchesExpression returns true if expression is empty.
// If the expression is not valid an error is logged and false is returned.
// if the tag is not nil, it evaluates the tag metadata against the expression.
// If the commit is not nil, it evaluates the commit metadata against the
// expression.
func (rc refreshConstraint) matchesExpressionFilter(ctx context.Context, sub *kargoapi.GitSubscription) bool {
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
func (rc refreshConstraint) matchesAllowIgnoreRules(ctx context.Context, sub *kargoapi.GitSubscription) bool {
	logger := logging.LoggerFromContext(ctx)
	if sub.AllowTags == "" {
		return true // no allow tags specified, so all tags are allowed
	}

	allowRegex, err := regexp.Compile(sub.AllowTags)
	if err != nil {
		logger.Debug("failed to compile allow regex",
			"tag", rc.Git.Tag,
			"allow", sub.AllowTags,
			"error", err.Error(),
		)
		return false
	}
	return warehouses.Ignores(rc.Git.Tag.Tag, sub.IgnoreTags) ||
		!warehouses.Allows(rc.Git.Tag.Tag, allowRegex)
}
