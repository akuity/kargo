package warehouses

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	libSemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/pattern"
)

const (
	regexpPrefix = "regexp:"
	regexPrefix  = "regex:"
	globPrefix   = "glob:"
)

// discoverCommits discovers commits from the given Git repositories based on the
// given subscriptions. It returns a list of GitDiscoveryResult objects, each
// containing the discovered commits for the corresponding subscription.
func (r *reconciler) discoverCommits(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.GitDiscoveryResult, error) {
	logger := logging.LoggerFromContext(ctx)

	results := make([]kargoapi.GitDiscoveryResult, 0, len(subs))

	repos := make([]git.Repo, 0, len(subs))
	defer func() {
		for _, repo := range repos {
			if err := repo.Close(); err != nil {
				logger.Error(
					err,
					"failed to clean up git repo",
					"repo", repo.URL(),
					"home", repo.HomeDir(),
					"path", repo.Dir(),
				)
			}
		}
	}()

	for _, s := range subs {
		if s.Git == nil {
			continue
		}

		sub := *s.Git

		repoLogger := logger.WithValues("repo", sub.RepoURL)

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
			repoLogger.Debug("obtained credentials for git repo")
		} else {
			repoLogger.Debug("found no credentials for git repo")
		}

		// Clone the Git repository.
		cloneOpts := &git.CloneOptions{
			Branch:       sub.Branch,
			SingleBranch: true,
			Filter:       git.FilterBlobless,
		}
		repo, err := r.gitCloneFn(
			sub.RepoURL,
			&git.ClientOptions{
				Credentials:           repoCreds,
				InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
			},
			cloneOpts,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to clone git repo %q: %w", sub.RepoURL, err)
		}
		// TODO: repos is a slice of repos that will be iterated and closed
		// (deleted) when this function returns. Implementations of r.gitCloneFn
		// used for testing sometimes return a nil repo since we don't have a mock
		// implementation for the git.Repo interface at present. With as many
		// methods as it has, it's a bit more expedient to just check that repo
		// isn't nil before adding it to the slice of repos to be closed.
		if repo != nil {
			repos = append(repos, repo)
		}

		// Enrich the logger with additional fields for this subscription.
		repoLogger = repoLogger.WithValues(gitDiscoveryLogFields(sub))

		// Discover commits based on the subscription's commit selection strategy.
		var discovered []kargoapi.DiscoveredCommit
		switch sub.CommitSelectionStrategy {
		case kargoapi.CommitSelectionStrategyLexical,
			kargoapi.CommitSelectionStrategyNewestTag,
			kargoapi.CommitSelectionStrategySemVer:
			tags, err := r.discoverTagsFn(repo, sub)
			if err != nil {
				return nil, fmt.Errorf("error listing tags from git repo %q: %w", sub.RepoURL, err)
			}

			for _, meta := range tags {
				discovered = append(discovered, kargoapi.DiscoveredCommit{
					ID:  meta.CommitID,
					Tag: meta.Tag,
					// A decent subject length for a commit message is 50 characters
					// (based on the 50/72 rule). We are nice people, and allow a
					// bit more. But not an excessive amount, to minimize the risk of
					// exceeding the maximum size of the object in the API server.
					Subject:     shortenString(meta.Subject, 80),
					Author:      meta.Author,
					Committer:   meta.Committer,
					CreatorDate: &metav1.Time{Time: meta.CreatorDate},
				})
				repoLogger.Trace(
					"discovered commit from tag",
					"tag", meta.Tag,
					"commit", meta.CommitID,
					"creatorDate", meta.CreatorDate.Format(time.RFC3339),
				)
			}
		default:
			commits, err := r.discoverBranchHistoryFn(repo, sub)
			if err != nil {
				return nil, fmt.Errorf("error listing commits from git repo %q: %w", sub.RepoURL, err)
			}

			for _, meta := range commits {
				discovered = append(discovered, kargoapi.DiscoveredCommit{
					ID:     meta.ID,
					Branch: sub.Branch,
					// A decent subject length for a commit message is 50 characters
					// (based on the 50/72 rule). We are nice people, and allow a
					// bit more. But not an excessive amount, to minimize the risk of
					// exceeding the maximum size of the object in the API server.
					Subject:     shortenString(meta.Subject, 80),
					Author:      meta.Author,
					Committer:   meta.Committer,
					CreatorDate: &metav1.Time{Time: meta.CommitDate},
				})
				repoLogger.Trace(
					"discovered commit from branch",
					"commit", meta.ID,
					"creatorDate", meta.CommitDate.Format(time.RFC3339),
				)
			}
		}

		if len(discovered) == 0 {
			results = append(results, kargoapi.GitDiscoveryResult{
				RepoURL: sub.RepoURL,
			})
			repoLogger.Debug("discovered no commits")
			continue
		}

		results = append(results, kargoapi.GitDiscoveryResult{
			RepoURL: sub.RepoURL,
			Commits: discovered,
		})
		repoLogger.Debug(
			"discovered commits",
			"count", len(discovered),
		)
	}

	return results, nil
}

// discoverBranchHistory returns a list of commits from the given Git repository
// that match the given subscription's branch selection criteria. It returns the
// list of commits that match the criteria, sorted in descending order. If the
// list contains more than 20 commits, it is clipped to the 20 most recent
// commits.
func (r *reconciler) discoverBranchHistory(repo git.Repo, sub kargoapi.GitSubscription) ([]git.CommitMetadata, error) {
	// Compile the commit expression filter if it is specified.
	var exprProgram *vm.Program
	if sub.ExpressionFilter != "" {
		program, err := expr.Compile(sub.ExpressionFilter)
		if err != nil {
			return nil, fmt.Errorf("error compiling expression filter: %w", err)
		}
		exprProgram = program
	}

	// Compile include and exclude path selectors.
	includeSelectors, err := getPathSelectors(sub.IncludePaths)
	if err != nil {
		return nil, fmt.Errorf("error parsing include selector: %w", err)
	}
	excludeSelectors, err := getPathSelectors(sub.ExcludePaths)
	if err != nil {
		return nil, fmt.Errorf("error parsing exclude selector: %w", err)
	}

	limit := int(sub.DiscoveryLimit)
	var filteredCommits = make([]git.CommitMetadata, 0, limit)
	for skip := uint(0); ; skip += uint(limit) { // nolint: gosec
		commits, err := r.listCommitsFn(repo, uint(limit), skip) // nolint: gosec
		if err != nil {
			return nil, fmt.Errorf("error listing commits from git repo %q: %w", sub.RepoURL, err)
		}

		// If no filters are specified, return the first commits up to the limit.
		if includeSelectors == nil && excludeSelectors == nil && exprProgram == nil {
			return commits, nil
		}

		// Filter commits based on include and exclude paths.
		for _, meta := range commits {
			// If the commit expression filter is specified, evaluate it.
			if exprProgram != nil {
				include, err := evaluateCommitExpression(meta, exprProgram)
				if err != nil {
					return nil, fmt.Errorf("error evaluating expression commit filter: %w", err)
				}
				if !include {
					continue
				}
			}

			// If include or exclude path selectors are specified, filter the commits.
			if includeSelectors != nil || excludeSelectors != nil {
				diffPaths, err := r.getDiffPathsForCommitIDFn(repo, meta.ID)
				if err != nil {
					return nil, fmt.Errorf(
						"error getting diff paths for commit %q in git repo %q: %w",
						meta.ID,
						sub.RepoURL,
						err,
					)
				}
				if !matchesPathsFilters(includeSelectors, excludeSelectors, diffPaths) {
					continue
				}
			}

			// If we reach this point, the commit matches the filters.
			filteredCommits = append(filteredCommits, meta)

			if len(filteredCommits) >= limit {
				return trimSlice(filteredCommits, limit), nil
			}
		}

		// If there are no more commits to list, break the loop.
		if len(commits) == 0 {
			break
		}
	}

	return trimSlice(filteredCommits, limit), nil
}

// discoverTags returns a list of tags from the given Git repository that match
// the given subscription's tag selection criteria. It returns the list of tags
// that match the criteria, sorted in descending order. If the list contains
// more than 20 tags, it is clipped to the 20 most recent tags.
func (r *reconciler) discoverTags(repo git.Repo, sub kargoapi.GitSubscription) ([]git.TagMetadata, error) {
	tags, err := r.listTagsFn(repo)
	if err != nil {
		return nil, fmt.Errorf("error listing tags from git repo %q: %w", sub.RepoURL, err)
	}

	if tags, err = filterTags(tags, sub.IgnoreTags, sub.AllowTags); err != nil {
		return nil, fmt.Errorf("failed to filter tags: %w", err)
	}

	if tags, err = filterTagsByExpression(tags, sub.ExpressionFilter); err != nil {
		return nil, fmt.Errorf("failed to filter tags by expression: %w", err)
	}

	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer:
		if tags, err = selectSemVerTags(tags, sub.StrictSemvers, sub.SemverConstraint); err != nil {
			return nil, fmt.Errorf("failed to select semver tags: %w", err)
		}
	case kargoapi.CommitSelectionStrategyLexical:
		slices.SortFunc(tags, func(i, j git.TagMetadata) int {
			// Sort in reverse lexicographic order
			return strings.Compare(j.Tag, i.Tag)
		})
	default:
		// No additional filtering or sorting required, as the tags are already
		// ordered by creation date.
	}

	// If no include or exclude paths are specified, return the first tags up to
	// the limit.
	limit := int(sub.DiscoveryLimit)
	if len(tags) == 0 || (sub.IncludePaths == nil && sub.ExcludePaths == nil) {
		return trimSlice(tags, limit), nil
	}

	// Compile include and exclude path selectors.
	includeSelectors, err := getPathSelectors(sub.IncludePaths)
	if err != nil {
		return nil, fmt.Errorf("error parsing include selector: %w", err)
	}
	excludeSelectors, err := getPathSelectors(sub.ExcludePaths)
	if err != nil {
		return nil, fmt.Errorf("error parsing exclude selector: %w", err)
	}

	// Filter tags based on include and exclude paths.
	var filteredTags = make([]git.TagMetadata, 0, limit)
	for _, meta := range tags {
		diffPaths, err := r.getDiffPathsForCommitIDFn(repo, meta.CommitID)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting diff paths for tag %q in git repo %q: %w",
				meta.Tag,
				sub.RepoURL,
				err,
			)
		}
		if matchesPathsFilters(includeSelectors, excludeSelectors, diffPaths) {
			filteredTags = append(filteredTags, meta)
		}

		if len(filteredTags) >= limit {
			break
		}
	}
	return trimSlice(filteredTags, limit), nil
}

// filterTags filters the given list of tag names based on the given allow and
// ignore criteria. It returns the filtered list of tag names.
func filterTags(tags []git.TagMetadata, ignoreTags []string, allow string) ([]git.TagMetadata, error) {
	allowRegex, err := regexp.Compile(allow)
	if err != nil {
		return nil, fmt.Errorf("error compiling regular expression %q: %w", allow, err)
	}
	filteredTags := make([]git.TagMetadata, 0, len(tags))
	for _, tag := range tags {
		if ignores(tag.Tag, ignoreTags) || !allows(tag.Tag, allowRegex) {
			continue
		}
		filteredTags = append(filteredTags, tag)
	}
	return slices.Clip(filteredTags), nil
}

// filterTagsByExpression filters the given list of tags based on the given
// expression. It returns the filtered list of tags. If the expression is empty,
// it returns the original list of tags.
//
// The expression is evaluated using the expr package, and the tag metadata is
// passed as the environment.
//
// For a tag to be included in the result, the expression must evaluate to true.
// If the expression evaluates to a non-boolean value, it is converted to a
// boolean using strconv.ParseBool. If the conversion fails, an error is
// returned.
func filterTagsByExpression(
	tags []git.TagMetadata,
	expression string,
) ([]git.TagMetadata, error) {
	if expression == "" {
		return tags, nil
	}

	program, err := expr.Compile(expression)
	if err != nil {
		return nil, fmt.Errorf("error compiling tag expression filter: %w", err)
	}

	filteredTags := make([]git.TagMetadata, 0, len(tags))
	for _, tag := range tags {
		env := map[string]any{
			"tag":         tag.Tag,
			"id":          tag.CommitID,
			"creatorDate": tag.CreatorDate,
			"author":      tag.Author,
			"committer":   tag.Committer,
			"subject":     tag.Subject,
		}

		result, err := expr.Run(program, env)
		if err != nil {
			return nil, fmt.Errorf("error evaluating tag expression filter: %w", err)
		}

		switch result := result.(type) {
		case bool:
			if !result {
				continue
			}
		default:
			parsedBool, err := strconv.ParseBool(fmt.Sprintf("%v", result))
			if err != nil {
				return nil, fmt.Errorf("error parsing expression result: %w", err)
			}
			if !parsedBool {
				continue
			}
		}

		filteredTags = append(filteredTags, tag)
	}
	return slices.Clip(filteredTags), nil
}

// evaluateCommitExpression evaluates the given commit expression against
// the given commit metadata. The commit metadata is passed as the environment
// for the expression evaluation. It returns true if the expression evaluates to
// true, and false otherwise. If the expression is not a boolean, it is
// converted to a boolean using strconv.ParseBool. If the conversion fails,
// an error is returned.
func evaluateCommitExpression(
	commit git.CommitMetadata,
	expression *vm.Program,
) (bool, error) {
	env := map[string]any{
		"id":         commit.ID,
		"commitDate": commit.CommitDate,
		"author":     commit.Author,
		"committer":  commit.Committer,
		"subject":    commit.Subject,
	}

	result, err := expr.Run(expression, env)
	if err != nil {
		return false, err
	}

	switch result := result.(type) {
	case bool:
		return result, nil
	default:
		parsedBool, err := strconv.ParseBool(fmt.Sprintf("%v", result))
		if err != nil {
			return false, err
		}
		return parsedBool, nil
	}
}

// allows returns true if the given tag name matches the given regular
// expression or if the regular expression is nil. It returns false otherwise.
func allows(tagName string, allowRegex *regexp.Regexp) bool {
	if allowRegex == nil {
		return true
	}
	return allowRegex.MatchString(tagName)
}

// ignores returns true if the given tag name is in the given list of ignored
// tag names. It returns false otherwise.
func ignores(tagName string, ignore []string) bool {
	for _, i := range ignore {
		if i == tagName {
			return true
		}
	}
	return false
}

func getPathSelectors(selectors []string) (pattern.Matcher, error) {
	if len(selectors) == 0 {
		return nil, nil
	}

	matchers := make(pattern.Matchers, len(selectors))
	for i := range selectors {
		matcher, err := pattern.ParsePathPattern(selectors[i])
		if err != nil {
			return nil, fmt.Errorf("parse error path selector %q: %w", selectors[i], err)
		}
		matchers[i] = matcher
	}
	return matchers, nil
}

func matchesPathsFilters(include, exclude pattern.Matcher, diffs []string) bool {
	for _, path := range diffs {
		// If include is nil, all paths are implicitly included
		// Otherwise, check if the path matches the include pattern
		if include != nil && !include.Matches(path) {
			// Path not included, skip to next path
			continue
		}

		// Path is included (either implicitly or explicitly)
		// Now check if it should be excluded
		if exclude != nil && exclude.Matches(path) {
			// Path is explicitly excluded, skip to next path
			continue
		}

		// If we reach here, the path is included and not excluded
		return true
	}

	// None of the paths match our criteria
	return false
}

func selectSemVerTags(tags []git.TagMetadata, strict bool, constraint string) ([]git.TagMetadata, error) {
	var svConstraint *semver.Constraints
	if constraint != "" {
		var err error
		if svConstraint, err = semver.NewConstraint(constraint); err != nil {
			return nil, fmt.Errorf("error parsing semver constraint %q: %w", constraint, err)
		}
	}

	type semVerTag struct {
		git.TagMetadata
		*semver.Version
	}

	var svs []semVerTag
	for _, meta := range tags {
		sv := libSemver.Parse(meta.Tag, strict)
		if sv == nil {
			continue
		}
		if svConstraint == nil || svConstraint.Check(sv) {
			svs = append(svs, semVerTag{
				TagMetadata: meta,
				Version:     sv,
			})
		}
	}

	slices.SortFunc(svs, func(i, j semVerTag) int {
		if comp := j.Compare(i.Version); comp != 0 {
			return comp
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., 1.0 and 1.0.0.
		return strings.Compare(j.Original(), i.Original())
	})

	var semverTags []git.TagMetadata
	for _, sv := range svs {
		semverTags = append(semverTags, sv.TagMetadata)
	}
	return semverTags, nil
}

func (r *reconciler) listCommits(repo git.Repo, limit, skip uint) ([]git.CommitMetadata, error) {
	return repo.ListCommits(limit, skip)
}

func (r *reconciler) listTags(repo git.Repo) ([]git.TagMetadata, error) {
	return repo.ListTags()
}

func (r *reconciler) getDiffPathsForCommitID(repo git.Repo, commitID string) ([]string, error) {
	return repo.GetDiffPathsForCommitID(commitID)
}

// gitDiscoveryLogFields returns a set of log fields for a Git subscription
// based on the subscription's configuration.
func gitDiscoveryLogFields(sub kargoapi.GitSubscription) []any {
	f := []any{
		"selectionStrategy", sub.CommitSelectionStrategy,
		"pathConstrained", sub.IncludePaths != nil || sub.ExcludePaths != nil,
	}
	if sub.Branch != "" {
		f = append(f, "branch", sub.Branch)
	}
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer:
		f = append(
			f,
			"semverConstraint", sub.SemverConstraint,
			"tagConstrained", sub.AllowTags != "" || len(sub.IgnoreTags) > 0,
		)
	case kargoapi.CommitSelectionStrategyLexical, kargoapi.CommitSelectionStrategyNewestTag:
		f = append(f, "tagConstrained", sub.AllowTags != "" || len(sub.IgnoreTags) > 0)
	}
	return f
}

// shortenString truncates the given string to the given length, appending an
// ellipsis if the string is longer than the length.
func shortenString(str string, length int) string {
	if length >= 0 && len(str) > length {
		return str[:length] + "..."
	}
	return str
}
