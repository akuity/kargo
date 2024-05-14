package warehouses

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	cloneOpts := &git.CloneOptions{
		Branch:                sub.Branch,
		SingleBranch:          true,
		Filter:                git.FilterBlobless,
		InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
	}
	repo, err := git.Clone(sub.RepoURL, &git.ClientOptions{Credentials: creds}, cloneOpts)
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

	var selectedTag, selectedCommit string
	var err error
	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategyNewestFromBranch:
		selectedCommit, err = r.getLastCommitIDFn(repo)
		if err != nil {
			return "", "",
				fmt.Errorf("error determining commit ID at head of branch %q in git repo %q: %w",
					sub.Branch,
					sub.RepoURL,
					err,
				)
		}
	case kargoapi.CommitSelectionStrategyLexical,
		kargoapi.CommitSelectionStrategyNewestTag,
		kargoapi.CommitSelectionStrategySemVer:
		selectedTags, err := r.discoverTagsFn(repo, sub)
		if err != nil {
			return "", "", fmt.Errorf("error discovering tags: %w", err)
		}
		if len(selectedTags) == 0 {
			return "", "", fmt.Errorf("found no applicable tags in repo %q", sub.RepoURL)
		}
		selectedTag = selectedTags[0].Tag
		selectedCommit = selectedTags[0].CommitID
	default:
		return "", "", fmt.Errorf("unknown commit selection strategy %q", sub.CommitSelectionStrategy)
	}

	// this shortcircuits to just return the last commit in case it is same as
	// baseCommit so we do not spam logs with errors of a valid not getting diffs
	// between baseCommit and HEAD (pointing to baseCommit in this case)
	if baseCommit == selectedCommit {
		return selectedTag, selectedCommit, nil
	}

	// In case includePaths/excludePaths filters are configured in a git subscription
	// below if clause deals with it. There is a special case - Warehouse has not
	// produced any Freight yet, this is sorted by creating Freight based on last
	// commit without applying filters.
	if (sub.IncludePaths != nil || sub.ExcludePaths != nil) && baseCommit != "" {

		// getting actual diffPaths since baseCommit
		diffs, err := r.getDiffPathsSinceCommitIDFn(repo, baseCommit)
		if err != nil {
			return selectedTag, "",
				fmt.Errorf("error getting diffs since commit %q in git repo %q: %w",
					baseCommit,
					sub.RepoURL,
					err,
				)
		}

		matchesPathsFilters, err := matchesPathsFilters(sub.IncludePaths, sub.ExcludePaths, diffs)
		if err != nil {
			return selectedTag, "",
				fmt.Errorf("error checking includePaths/excludePaths match for commit %q for git repo %q: %w",
					selectedCommit,
					sub.RepoURL,
					err,
				)
		}

		if !matchesPathsFilters {
			return selectedTag, "",
				fmt.Errorf("commit %q not applicable due to includePaths/excludePaths configuration for repo %q",
					selectedCommit,
					sub.RepoURL,
				)
		}
	}

	return selectedTag, selectedCommit, nil
}

func (r *reconciler) discoverCommits(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.GitDiscoveryResult, error) {
	results := make([]kargoapi.GitDiscoveryResult, 0, len(subs))

	for _, s := range subs {
		if s.Git == nil {
			continue
		}

		sub := *s.Git

		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)

		creds, ok, err := r.credentialsDB.Get(ctx, namespace, credentials.TypeGit, sub.RepoURL)
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

		cloneOpts := &git.CloneOptions{
			Branch:                sub.Branch,
			SingleBranch:          true,
			Filter:                git.FilterBlobless,
			InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
		}

		repo, err := git.Clone(
			sub.RepoURL,
			&git.ClientOptions{
				Credentials: repoCreds,
			},
			cloneOpts,
		)
		if err != nil {
			return nil, fmt.Errorf("error cloning git repo %q: %w", sub.RepoURL, err)
		}

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
					ID:        meta.CommitID,
					Tag:       meta.Tag,
					Subject:   meta.Subject,
					CreatedAt: metav1.Time{Time: meta.CreatorDate},
				})
			}
		default:
			commits, err := r.discoverBranchHistoryFn(repo, sub)
			if err != nil {
				return nil, fmt.Errorf("error listing commits from git repo %q: %w", sub.RepoURL, err)
			}

			for _, meta := range commits {
				discovered = append(discovered, kargoapi.DiscoveredCommit{
					ID:        meta.ID,
					Branch:    sub.Branch,
					Subject:   meta.Subject,
					CreatedAt: metav1.Time{Time: meta.CommitDate},
				})
			}
		}

		results = append(results, kargoapi.GitDiscoveryResult{
			RepoURL: sub.RepoURL,
			Commits: discovered,
		})
	}

	return results, nil
}

func (r *reconciler) discoverBranchHistory(repo git.Repo, sub kargoapi.GitSubscription) ([]git.CommitMetadata, error) {
	const limit = 20

	var filteredCommits []git.CommitMetadata
	for skip := uint(0); ; skip += limit {
		commits, err := r.listCommitsWithMetadataFn(repo, limit, skip)
		if err != nil {
			return nil, fmt.Errorf("error listing commits from git repo %q: %w", sub.RepoURL, err)
		}

		// If no include or exclude paths are specified, return the first commits
		// up to the limit.
		if sub.IncludePaths == nil && sub.ExcludePaths == nil {
			return commits, nil
		}

		if filteredCommits == nil {
			filteredCommits = make([]git.CommitMetadata, 0, limit)
		}

		// Filter commits based on include and exclude paths.
		for _, meta := range commits {
			diffPaths, err := r.getDiffPathsForCommitIDFn(repo, meta.ID)
			if err != nil {
				return nil, fmt.Errorf(
					"error getting diff paths for commit %q in git repo %q: %w",
					meta.ID,
					sub.RepoURL,
					err,
				)
			}
			matchesPathsFilters, err := matchesPathsFilters(sub.IncludePaths, sub.ExcludePaths, diffPaths)
			if err != nil {
				return nil, fmt.Errorf(
					"error checking includePaths/excludePaths match for commit %q for git repo %q: %w",
					meta.ID,
					sub.RepoURL,
					err,
				)
			}
			if matchesPathsFilters {
				filteredCommits = append(filteredCommits, meta)
			}

			if len(filteredCommits) >= limit {
				return filteredCommits, nil
			}
		}

		// If there are no more commits to list, break the loop.
		if len(commits) == 0 {
			break
		}
	}

	return slices.Clip(filteredCommits), nil
}

// discoverTags returns a list of tags from the given Git repository that match
// the given subscription's tag selection criteria. It returns the list of tags
// that match the criteria, sorted in descending order. If the list contains
// more than 20 tags, it is clipped to the 20 most recent tags.
func (r *reconciler) discoverTags(repo git.Repo, sub kargoapi.GitSubscription) ([]git.TagMetadata, error) {
	tags, err := r.listTagsWithMetadataFn(repo)
	if err != nil {
		return nil, fmt.Errorf("error listing tags from git repo %q: %w", sub.RepoURL, err)
	}

	if tags, err = filterTags(tags, sub.IgnoreTags, sub.AllowTags); err != nil {
		return nil, fmt.Errorf("failed to filter tags: %w", err)
	}

	switch sub.CommitSelectionStrategy {
	case kargoapi.CommitSelectionStrategySemVer:
		if tags, err = selectSemVerTags(tags, sub.SemverConstraint); err != nil {
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

	if l := len(tags); l < 20 {
		return tags, nil
	}
	return tags[:20], nil
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

func selectSemVerTags(tags []git.TagMetadata, constraint string) ([]git.TagMetadata, error) {
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
		sv, err := semver.NewVersion(meta.Tag)
		if err != nil {
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

func (r *reconciler) getLastCommitID(repo git.Repo) (string, error) {
	return repo.LastCommitID()
}

func (r *reconciler) listCommitsWithMetadata(repo git.Repo, limit, skip uint) ([]git.CommitMetadata, error) {
	return repo.ListCommitsWithMetadata(limit, skip)
}

func (r *reconciler) listTagsWithMetadata(repo git.Repo) ([]git.TagMetadata, error) {
	return repo.ListTagsWithMetadata()
}

func (r *reconciler) getDiffPathsSinceCommitID(repo git.Repo, commitId string) ([]string, error) {
	return repo.GetDiffPathsSinceCommitID(commitId)
}

func (r *reconciler) getDiffPathsForCommitID(repo git.Repo, commitID string) ([]string, error) {
	return repo.GetDiffPathsForCommitID(commitID)
}
