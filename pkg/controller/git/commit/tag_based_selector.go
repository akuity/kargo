package commit

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/logging"
)

const tagPrefix = "refs/tags/"

// tagBasedSelector is a base implementation of Selector that provides common
// functionality for all Selector implementations that select commits on the
// basis of tag names or metadata. It is not intended to be used directly.
type tagBasedSelector struct {
	*baseSelector
	allowTagsRegexes  []*regexp.Regexp
	ignoreTagsRegexes []*regexp.Regexp

	filterTagsByDiffPathsFn func(
		git.Repo,
		[]git.TagMetadata,
	) ([]git.TagMetadata, error)
}

// compileRegexes returns a slice of compiled regular expressions.
func compileRegexes(regexStrs []string) ([]*regexp.Regexp, error) {
	regexes := make([]*regexp.Regexp, len(regexStrs))
	var err error
	for i, regexStr := range regexStrs {
		if regexes[i], err = regexp.Compile(regexStr); err != nil {
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				regexStr, err,
			)
		}
	}
	return regexes, nil
}

func newTagBasedSelector(
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (*tagBasedSelector, error) {
	base, err := newBaseSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	s := &tagBasedSelector{
		baseSelector: base,
	}

	if s.allowTagsRegexes, err = compileRegexes(sub.AllowTagsRegexes); err != nil {
		return nil, err
	}

	// TODO(v1.11.0): Return an error if sub.AllowTags is non-empty.
	// TODO(v1.13.0): Remove this block after the AllowTags field is removed.
	if sub.AllowTags != "" { // nolint: staticcheck
		var allowTagsRegex *regexp.Regexp
		if allowTagsRegex, err = regexp.Compile(sub.AllowTags); err != nil { // nolint: staticcheck
			return nil, fmt.Errorf(
				"error compiling regular expression %q: %w",
				sub.AllowTags, err, // nolint: staticcheck
			)
		}
		s.allowTagsRegexes = append(s.allowTagsRegexes, allowTagsRegex)
	}

	if s.ignoreTagsRegexes, err = compileRegexes(sub.IgnoreTagsRegexes); err != nil {
		return nil, err
	}

	// TODO(v1.11.0): Return an error if sub.IgnoreTags is non-empty.
	// TODO(v1.13.0): Remove this block after the IgnoreTags field is removed.
	if len(sub.IgnoreTags) > 0 { // nolint: staticcheck
		ignoreTagsRegexStrs := make([]string, len(sub.IgnoreTags)) // nolint: staticcheck
		for i, ignoreTag := range sub.IgnoreTags {                 // nolint: staticcheck
			ignoreTagsRegexStrs[i] = fmt.Sprintf("^%s$", regexp.QuoteMeta(ignoreTag))
		}
		ignoreTagsRegexes, err := compileRegexes(ignoreTagsRegexStrs)
		if err != nil {
			return nil, err
		}
		s.ignoreTagsRegexes = append(s.ignoreTagsRegexes, ignoreTagsRegexes...)
	}

	s.filterTagsByDiffPathsFn = s.filterTagsByDiffPaths
	return s, nil
}

// MatchesRef implements Selector.
func (t *tagBasedSelector) MatchesRef(ref string) bool {
	if !strings.HasPrefix(ref, tagPrefix) {
		return false
	}
	return t.matchesTag(ref)
}

// getLoggerContext returns key/value pairs that can be used by any selector
// that selects commits on the basis of tag names or metadata to enrich loggers
// with valuable context.
func (t *tagBasedSelector) getLoggerContext() []any {
	return append(
		t.baseSelector.getLoggerContext(),
		"tagConstrained", len(t.allowTagsRegexes) > 0 || len(t.ignoreTagsRegexes) > 0,
	)
}

// matchesTag returns a boolean indicating whether the given tag satisfies the
// selector's constraints. Any leading "refs/tags/" is stripped away prior
// to evaluation.
func (t *tagBasedSelector) matchesTag(tag string) bool {
	tag = strings.TrimPrefix(tag, tagPrefix)

	// handle ignoreTagsRegexes
	for _, regex := range t.ignoreTagsRegexes {
		if regex.MatchString(tag) {
			return false
		}
	}

	// if empty allowTagsRegexes, we match all tags
	if len(t.allowTagsRegexes) == 0 {
		return true
	}

	// check if tag matches any allowTagsRegexes
	for _, regex := range t.allowTagsRegexes {
		if regex.MatchString(tag) {
			return true
		}
	}

	return false
}

// clone clones a Git repository specified by the selector's repoURL field using
// options suitable for selectors that selects commits on the basis of tag names
// or metadata.
func (t *tagBasedSelector) clone(ctx context.Context) (git.Repo, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("cloning repository")
	cloneOpts := &git.CloneOptions{
		SingleBranch: true,
		Filter:       git.FilterBlobless,
	}
	repo, err := t.gitCloneFn(
		t.repoURL,
		&git.ClientOptions{
			Credentials:           t.creds,
			InsecureSkipTLSVerify: t.insecureSkipTLSVerify,
		},
		cloneOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("error cloning git repo %q: %w", t.repoURL, err)
	}
	return repo, nil
}

// filterTags evaluates all provided tags against the constraints defined by the
// t.matchesTag method, returning only those that satisfied those constraints.
func (t *tagBasedSelector) filterTags(tags []git.TagMetadata) []git.TagMetadata {
	filteredTags := make([]git.TagMetadata, 0, len(tags))
	for _, tag := range tags {
		if t.matchesTag(tag.Tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	return slices.Clip(filteredTags)
}

// filterTagsByExpression evaluates the metadata of all provided tags against
// user-defined expressions, returning only those tags whose metadata satisfied
// those constraints.
func (t *tagBasedSelector) filterTagsByExpression(
	tags []git.TagMetadata,
) ([]git.TagMetadata, error) {
	if t.filterExpression == nil {
		return tags, nil
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
			"tagger":      tag.Tagger,
			"annotation":  tag.Annotation,
		}

		result, err := expr.Run(t.filterExpression, env)
		if err != nil {
			return nil, fmt.Errorf("error evaluating tag filter expression: %w", err)
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

// filterTagsByDiffPaths iterates over all provided tags, for each, retrieving
// information about paths affected by the commit it references and evaluating
// those paths against user-defined path-selection criteria. Only tags pointing
// to commits that satisfy those criteria are returned.
func (t *tagBasedSelector) filterTagsByDiffPaths(
	repo git.Repo,
	tags []git.TagMetadata,
) ([]git.TagMetadata, error) {
	if len(tags) == 0 || (t.includePaths == nil && t.excludePaths == nil) {
		return tags, nil
	}
	filteredTags := make([]git.TagMetadata, 0, t.discoveryLimit)
	for _, tag := range tags {
		diffPaths, err := repo.GetDiffPathsForCommitID(tag.CommitID)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting diff paths for tag %q in git repo %q: %w",
				tag.Tag,
				t.repoURL,
				err,
			)
		}
		if matchesPathsFilters(t.includePaths, t.excludePaths, diffPaths) {
			filteredTags = append(filteredTags, tag)
		}
		if len(filteredTags) >= t.discoveryLimit {
			break
		}
	}
	return filteredTags, nil
}

// tagsToAPICommits converts a slice of git.TagMetadata to a slice of
// kargoapi.DiscoveredCommit, which can be directly used by a caller performing
// artifact discovery. If the number of tags provided exceeds the selector's
// discovery limit, the slice returned will be truncated so as not to exceed
// that limit.
func (t *tagBasedSelector) tagsToAPICommits(
	ctx context.Context,
	tags []git.TagMetadata,
) []kargoapi.DiscoveredCommit {
	logger := logging.LoggerFromContext(ctx)
	tags = trimSlice(tags, t.discoveryLimit)
	commits := make([]kargoapi.DiscoveredCommit, len(tags))
	for i, tag := range tags {
		commits[i] = kargoapi.DiscoveredCommit{
			ID:  tag.CommitID,
			Tag: tag.Tag,
			// A decent subject length for a commit message is 50 characters
			// (based on the 50/72 rule). We are nice people, and allow a
			// bit more. But not an excessive amount, to minimize the risk of
			// exceeding the maximum size of the object in the API server.
			Subject:     shortenString(tag.Subject, 80),
			Author:      tag.Author,
			Committer:   tag.Committer,
			CreatorDate: &metav1.Time{Time: tag.CreatorDate},
		}
		logger.Trace(
			"discovered commit from tag",
			"tag", tag.Tag,
			"commit", tag.CommitID,
			"creatorDate", tag.CreatorDate.Format(time.RFC3339),
		)
	}
	logger.Debug("discovered commits", "count", len(commits))
	return commits
}
