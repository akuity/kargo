package warehouses

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

const (
	regexpPrefix = "regexp:"
	regexPrefix  = "regex:"
	globPrefix   = "glob:"
)

type gitMeta struct {
	Commit  string
	Tag     string
	Message string
	Author  string
}

type pathSelector func(path string) (bool, error)

func (r *reconciler) selectCommits(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
	lastFreight *kargoapi.FreightReference,
) ([]kargoapi.GitCommit, error) {
	latestCommits := make([]kargoapi.GitCommit, 0, len(subs))

	var repoCommitMappings map[string]string
	if lastFreight != nil {
		repoCommitMappings = make(map[string]string, len(lastFreight.Commits))
		for _, commit := range lastFreight.Commits {
			repoCommitMappings[commit.RepoURL+"#"+commit.Branch] = commit.ID
		}
	}

	for _, s := range subs {
		if s.Git == nil {
			continue
		}
		sub := s.Git
		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)
		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, sub.RepoURL)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining credentials for git repo %q: %w",
				sub.RepoURL,
				err,
			)
		}
		var repoCreds *git.RepoCredentials
		if ok {
			repoCreds = &git.RepoCredentials{
				Username:      creds.Username,
				Password:      creds.Password,
				SSHPrivateKey: creds.SSHPrivateKey,
			}
			logger.Debug("obtained credentials for git repo")
		} else {
			logger.Debug("found no credentials for git repo")
		}

		baseCommit := repoCommitMappings[sub.RepoURL+"#"+sub.Branch]

		gm, err := r.selectCommitMetaFn(ctx, *s.Git, repoCreds, baseCommit)
		if err != nil {
			return nil, fmt.Errorf(
				"error determining latest commit ID of git repo %q: %w",
				sub.RepoURL,
				err,
			)
		}
		logger.WithField("commit", gm.Commit).
			Debug("found latest commit from repo")
		latestCommits = append(
			latestCommits,
			kargoapi.GitCommit{
				RepoURL: sub.RepoURL,
				ID:      gm.Commit,
				Branch:  sub.Branch,
				Tag:     gm.Tag,
				Message: gm.Message,
			},
		)
	}
	return latestCommits, nil
}

// selectCommitMeta uses criteria from the provided GitSubscription to select
// an appropriate revision of the repository also specified by the subscription
// and return metadata associated with that revision.
func (r *reconciler) selectCommitMeta(
	ctx context.Context,
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
	baseCommit string,
) (*gitMeta, error) {
	logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	if sub.CommitSelectionStrategy == "" {
		sub.CommitSelectionStrategy = kargoapi.CommitSelectionStrategyNewestFromBranch
	}
	// when includePaths and/or excludePaths filters are used we can't use shallow clone
	// as we need diffs between HEAD and a baseCommit which depth in git history is unknown
	var shallowClone = true
	if (len(sub.IncludePaths) != 0 || len(sub.ExcludePaths) != 0) && baseCommit != "" {
		shallowClone = false
	}
	repo, err := git.Clone(
		sub.RepoURL,
		&git.ClientOptions{
			Credentials: creds,
		},
		&git.CloneOptions{
			Branch:                sub.Branch,
			SingleBranch:          true,
			Shallow:               shallowClone,
			InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error cloning git repo %q: %w", sub.RepoURL, err)
	}
	selectedTag, selectedCommit, err := r.selectTagAndCommitID(repo, sub, baseCommit)
	if err != nil {
		return nil, fmt.Errorf(
			"error selecting commit from git repo %q: %w",
			sub.RepoURL,
			err,
		)
	}
	msg, err := repo.CommitMessage(selectedCommit)
	if err != nil {
		// This is best effort, so just log the error
		logger.Warnf("failed to get message from commit %q: %v", selectedCommit, err)
	}
	return &gitMeta{
		Commit: selectedCommit,
		Tag:    selectedTag,
		// Since we currently store commit messages in Stage status, we only capture
		// the first line of the commit message for brevity
		Message: strings.Split(strings.TrimSpace(msg), "\n")[0],
		// TODO: support git author
	}, nil
}

// selectTagAndCommitID uses criteria from the provided GitSubscription to
// select and return an appropriate revision of the repository also specified by
// the subscription.
func (r *reconciler) selectTagAndCommitID(
	repo git.Repo,
	sub kargoapi.GitSubscription,
	baseCommit string,
) (string, string, error) {

	continueWithoutTag := false
	var filteredTags []string
	var err error
	// get filtered list of repo tags only if strategy differs from NewestFromBranch
	if sub.CommitSelectionStrategy != kargoapi.CommitSelectionStrategyNewestFromBranch {
		filteredTags, err = r.getFilteredTags(repo, sub)
		if err != nil {
			return "", "", fmt.Errorf("error getting applicable tags: %w", err)
		}
	}

	var selectedTag string
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategyNewestFromBranch:
		// set flag for further processing to continue with empty selectedTag, that
		// is a special case with NewestFromBranch strategy
		continueWithoutTag = true
	case kargoapi.CommitSelectionStrategyLexical:
		selectedTag = selectLexicallyLastTag(filteredTags)
	case kargoapi.CommitSelectionStrategyNewestTag:
		selectedTag = filteredTags[0] // These are already ordered newest to oldest
	case kargoapi.CommitSelectionStrategySemVer:
		if selectedTag, err =
			selectSemverTag(filteredTags, sub.SemverConstraint); err != nil {
			return "", "", err
		}
	default:
		return "", "", fmt.Errorf("unknown commit selection strategy %q", sub.CommitSelectionStrategy)
	}

	if selectedTag != "" {
		// Checkout the tag only if we actually have selected a valid one
		if err = r.checkoutTagFn(repo, selectedTag); err != nil {
			return "", "", fmt.Errorf(
				"error checking out tag %q from git repo %q: %w",
				selectedTag,
				sub.RepoURL,
				err,
			)
		}
	} else if !continueWithoutTag {
		// we only get here when one of tag handling strategies is used and
		// selection process did not find a valid tag
		return "", "", fmt.Errorf("found no applicable tags in repo %q", sub.RepoURL)
	}

	// this now is common commit determining functionality for
	// both cases - one of tag handling strategies and NewestFromBranch strategy
	commit, err := r.getLastCommitIDFn(repo)
	if err != nil {
		return "", "", fmt.Errorf(
			"error determining commit ID in git repo %q: %w",
			sub.RepoURL,
			err,
		)

	}

	if err := r.verifyPathFiltersDoSelectCommitOtherwiseReturnError(repo, sub, commit, baseCommit); err != nil {
		return "", "", fmt.Errorf("commit not applicable due to path filters error: %w", err)
	}

	return selectedTag, commit, nil

}

// getFilteredTags gets and narrows down the list of repository tags according
// to the tag filtering configuration in Git subscription
func (r *reconciler) getFilteredTags(repo git.Repo, sub kargoapi.GitSubscription) ([]string, error) {
	tags, err := r.listTagsFn(repo) // These are ordered newest to oldest
	if err != nil {
		return nil, fmt.Errorf("error listing tags from git repo %q: %w", sub.RepoURL, err)
	}

	// Narrow down the list of tags to those that are allowed and not ignored
	allowRegex, err := regexp.Compile(sub.AllowTags)
	if err != nil {
		return nil, fmt.Errorf("error compiling regular expression %q: %w", sub.AllowTags, err)
	}
	filteredTags := make([]string, 0, len(tags))
	for _, tagName := range tags {
		if allows(tagName, allowRegex) && !ignores(tagName, sub.IgnoreTags) {
			filteredTags = append(filteredTags, tagName)
		}
	}
	if len(filteredTags) == 0 {
		return nil, fmt.Errorf("found no applicable tags in repo %q", sub.RepoURL)
	}
	return filteredTags, nil
}

// verifyPathFiltersDoSelectCommitOtherwiseReturnError applies path filters
// configuration matching if applicable and returns nil if it succeeded to
// select the commit, otherwise it returns the downstream error
func (r *reconciler) verifyPathFiltersDoSelectCommitOtherwiseReturnError(
	repo git.Repo,
	sub kargoapi.GitSubscription,
	commit string,
	baseCommit string,
) error {
	// In case includePaths/excludePaths filters are configured in a git subscription
	// below if clause deals with it. There is a special case - Warehouse has not
	// produced any Freight yet, this is sorted by creating Freight based on last
	// commit without applying filters.
	if (sub.IncludePaths != nil || sub.ExcludePaths != nil) && baseCommit != "" {

		// this shortcircuits to just return the last commit in case it is same as
		// baseCommit so we do not spam logs with errors of a valid not getting diffs
		// between baseCommit and HEAD (pointing to baseCommit in this case)
		if baseCommit == commit {
			return nil
		}

		// getting actual diffPaths since baseCommit
		diffs, err := r.getDiffPathsSinceCommitIDFn(repo, baseCommit)
		if err != nil {
			return fmt.Errorf("error getting diffs since commit %q in git repo %q: %w",
				baseCommit,
				sub.RepoURL,
				err,
			)
		}

		matchesPathsFilters, err := matchesPathsFilters(sub.IncludePaths, sub.ExcludePaths, diffs)
		if err != nil {
			return fmt.Errorf("error checking includePaths/excludePaths match for commit %q for git repo %q: %w",
				commit,
				sub.RepoURL,
				err,
			)
		}

		if !matchesPathsFilters {
			return fmt.Errorf("commit %q not applicable due to includePaths/excludePaths configuration for repo %q",
				commit,
				sub.RepoURL,
			)
		}
	}
	return nil
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

func getPathSelectors(selectorStrs []string) ([]pathSelector, error) {
	selectors := make([]pathSelector, len(selectorStrs))
	for i, selectorStr := range selectorStrs {
		switch {
		case strings.HasPrefix(selectorStr, regexpPrefix):
			regex, err := regexp.Compile(strings.TrimPrefix(selectorStr, regexpPrefix))
			if err != nil {
				return nil, err
			}
			selectors[i] = func(path string) (bool, error) {
				return regex.MatchString(path), nil
			}
		case strings.HasPrefix(selectorStr, regexPrefix):
			regex, err := regexp.Compile(strings.TrimPrefix(selectorStr, regexPrefix))
			if err != nil {
				return nil, err
			}
			selectors[i] = func(path string) (bool, error) {
				return regex.MatchString(path), nil
			}
		case strings.HasPrefix(selectorStr, globPrefix):
			pattern := strings.TrimPrefix(selectorStr, globPrefix)
			selectors[i] = func(path string) (bool, error) {
				return filepath.Match(pattern, path)
			}
		default:
			basePath := selectorStr
			selectors[i] = func(path string) (bool, error) {
				relPath, err := filepath.Rel(basePath, path)
				if err != nil {
					return false, err
				}
				return !strings.Contains(relPath, ".."), nil
			}
		}
	}
	return selectors, nil
}

func matchesPathsFilters(includePaths []string, excludePaths []string, diffs []string) (bool, error) {
	includeSelectors, err := getPathSelectors(includePaths)
	if err != nil {
		return false, err
	}
	excludeSelectors, err := getPathSelectors(excludePaths)
	if err != nil {
		return false, err
	}
pathLoop:
	for _, path := range diffs {
		if len(includeSelectors) > 0 {
			var selected bool
			for _, selector := range includeSelectors {
				if selected, err = selector(path); err != nil {
					return false, err
				}
				if selected {
					// Path was explicitly included, so we can move on to checking if
					// it should be excluded
					break
				}
			}
			if !selected {
				// Path was not explicitly included, so we can move on to the next path
				continue pathLoop
			}
		}
		// If we reach this point, the path was either implicitly or explicitly
		// included. Now check if it should be excluded.
		for _, selector := range excludeSelectors {
			selected, err := selector(path)
			if err != nil {
				return false, err
			}
			if selected {
				// Path was explicitly excluded, so we can move on to the next path
				continue pathLoop
			}
		}
		// If we reach this point, the path was not explicitly excluded
		return true, nil
	}
	return false, nil
}

// selectLexicallyLastTag sorts the provided tag name in reverse lexicographic
// order and returns the first tag name in the sorted list. If the list is
// empty, it returns an empty string.
func selectLexicallyLastTag(tagNames []string) string {
	if len(tagNames) == 0 {
		return ""
	}
	sort.Slice(tagNames, func(i, j int) bool {
		return tagNames[i] > tagNames[j]
	})
	return tagNames[0]
}

// selectSemverTag narrows the provided list of tag names to those that are
// valid semantic versions. If constraintStr is non-empty, it further narrows
// the list to those that satisfy the constraint. If the narrowed list is
// non-empty, it sorts the list in reverse semver order and returns the first
// tag name in the sorted list. If the narrowed list is empty, it returns an
// empty string.
func selectSemverTag(tagNames []string, constraintStr string) (string, error) {
	var constraint *semver.Constraints
	if constraintStr != "" {
		var err error
		if constraint, err = semver.NewConstraint(constraintStr); err != nil {
			return "", fmt.Errorf(
				"error parsing semver constraint %q: %w",
				constraintStr,
				err,
			)
		}
	}
	semvers := make([]*semver.Version, 0, len(tagNames))
	for _, tagName := range tagNames {
		sv, err := semver.NewVersion(tagName)
		if err != nil {
			continue // tagName wasn't a semantic version
		}
		if constraint == nil || constraint.Check(sv) {
			semvers = append(semvers, sv)
		}
	}
	if len(semvers) == 0 {
		return "", nil
	}
	sort.Slice(semvers, func(i, j int) bool {
		if comp := semvers[i].Compare(semvers[j]); comp != 0 {
			return comp > 0
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison
		// of equivalent semvers, e.g., 1.0 and 1.0.0.
		return semvers[i].Original() > semvers[j].Original()
	})
	return semvers[0].Original(), nil
}

func (r *reconciler) getLastCommitID(repo git.Repo) (string, error) {
	return repo.LastCommitID()
}

func (r *reconciler) listTags(repo git.Repo) ([]string, error) {
	return repo.ListTags()
}

func (r *reconciler) checkoutTag(repo git.Repo, tag string) error {
	return repo.Checkout(tag)
}

func (r *reconciler) getDiffPathsSinceCommitID(repo git.Repo, commitId string) ([]string, error) {
	return repo.GetDiffPathsSinceCommitID(commitId)
}
