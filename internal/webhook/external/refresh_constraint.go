package external

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"

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
		Tag        string
		Branch     string
		Diffs      []string
		Expression string
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
	return slices.ContainsFunc(
		filterSubsByRepoURL(subs, repoURLs...),
		func(sub kargoapi.RepoSubscription) bool {
			return rc.matches(ctx, sub)
		},
	)
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

func (rc refreshConstraint) matchesGitConstraint(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
) bool {
	if rc.Git == nil || sub == nil {
		return false
	}
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategyLexical:
		return rc.matchesLexicalConstraint(ctx, rc.Git.Tag, sub)
	case kargoapi.CommitSelectionStrategySemVer:
		return rc.matchesSemVerConstraint(ctx, sub)
	case kargoapi.CommitSelectionStrategyNewestTag:
		return rc.matchesNewestTagConstraint(ctx, sub)
	default:
		return rc.matchesNewestFromBranchConstraint(ctx, sub)
	}
}

func (rc refreshConstraint) matchesSemVerConstraint(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
) bool {
	logger := logging.LoggerFromContext(ctx)

	constraint, err := semver.NewConstraint(sub.SemverConstraint)
	if err != nil {
		logger.Debug("failed to parse semver constraint",
			"tag", rc.Git.Tag,
			"constraint", sub.SemverConstraint,
			"error", err.Error(),
		)
		return false
	}

	version := libSemver.Parse(rc.Git.Tag, sub.StrictSemvers)
	if version == nil {
		logger.Debug("tag is not semver formatted",
			"tag", rc.Git.Tag,
			"constraint", sub.SemverConstraint,
			"strictSemvers", sub.StrictSemvers,
		)
		return false
	}

	if !constraint.Check(version) {
		logger.Debug("tag does not satisfy semver constraint",
			"tag", rc.Git.Tag,
			"constraint", sub.SemverConstraint,
		)
		return false
	}
	return rc.matchesBaseFilters(
		ctx,
		sub,
		rc.Git.Expression,
		git.TagMetadata{
			// TODO(Faris): Fill this in with actual tag metadata
		},
	)
}

func (rc refreshConstraint) matchesLexicalConstraint(
	ctx context.Context,
	tag string,
	sub *kargoapi.GitSubscription,
) bool {
	logger := logging.LoggerFromContext(ctx)
	allowRegex, err := regexp.Compile(sub.AllowTags)
	if err != nil {
		logger.Debug("failed to compile allow regex",
			"tag", tag,
			"allow", sub.AllowTags,
			"error", err.Error(),
		)
		return false
	}
	if warehouses.Ignores(tag, sub.IgnoreTags) || !warehouses.Allows(tag, allowRegex) {
		return false
	}
	return rc.matchesBaseFilters(
		ctx,
		sub,
		rc.Git.Expression,
		git.TagMetadata{
			// TODO(Faris): Fill this in with actual tag metadata
		},
	)
}

func (rc refreshConstraint) matchesNewestFromBranchConstraint(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
) bool {
	if rc.Git.Branch != sub.Branch {
		return false
	}
	return rc.matchesBaseFilters(
		ctx,
		sub,
		rc.Git.Expression,
		git.TagMetadata{},
	)
}

func (rc refreshConstraint) matchesNewestTagConstraint(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
) bool {
	// TODO(Faris): evaluates if tag is newest

	return rc.matchesBaseFilters(
		ctx,
		sub,
		rc.Git.Expression,
		git.TagMetadata{},
	)
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

func matchesPathFilters(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
	diffPaths []string,
) bool {
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
		diffPaths,
	)
}

func matchesExpression(
	ctx context.Context,
	expression string,
	tag git.TagMetadata,
	value string,
) bool {
	if expression == "" {
		return true
	}

	logger := logging.LoggerFromContext(ctx).WithValues(
		"expression", expression,
		"value", value,
	)

	program, err := expr.Compile(expression)
	if err != nil {
		logger.Error(err, "error compiling tag expression filter")
		return false
	}

	env := map[string]any{
		"tag":         tag.Tag,
		"id":          tag.CommitID,
		"creatorDate": tag.CreatorDate,
		"author":      tag.Author,
		"committer":   tag.Committer,
		"subject":     tag.Subject,
		"tagger":      tag.Tagger,
		"annotation":  tag.Annotation,
	}

	result, err := expr.Run(program, env)
	if err != nil {
		logger.Error(err, "error evaluating tag expression filter")
		return false
	}

	switch result := result.(type) {
	case bool:
		return result
	default:
		parsedBool, err := strconv.ParseBool(fmt.Sprintf("%v", result))
		if err != nil {
			logger.Error(err, "error parsing expression result")
			return false
		}
		return parsedBool
	}
}

func (rc refreshConstraint) matchesBaseFilters(
	ctx context.Context,
	sub *kargoapi.GitSubscription,
	expression string,
	tag git.TagMetadata,
) bool {
	return matchesPathFilters(ctx, sub, rc.Git.Diffs) &&
		matchesExpression(ctx,
			sub.ExpressionFilter,
			git.TagMetadata{
				// TODO(Faris): Fill this in with actual tag metadata
			},
			rc.Git.Expression,
		)
}
