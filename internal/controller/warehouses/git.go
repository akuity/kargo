package warehouses

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

type gitMeta struct {
	Commit  string
	Tag     string
	Message string
	Author  string
}

func (r *reconciler) selectCommits(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.GitCommit, error) {
	latestCommits := make([]kargoapi.GitCommit, 0, len(subs))
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

		gm, err := r.selectCommitMetaFn(ctx, *s.Git, repoCreds)
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
) (*gitMeta, error) {
	logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)
	if creds == nil {
		creds = &git.RepoCredentials{}
	}
	if sub.CommitSelectionStrategy == "" {
		sub.CommitSelectionStrategy = kargoapi.CommitSelectionStrategyNewestFromBranch
	}
	repo, err := git.Clone(
		sub.RepoURL,
		*creds,
		&git.CloneOptions{
			Branch:                sub.Branch,
			SingleBranch:          true,
			Shallow:               true,
			InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error cloning git repo %q: %w", sub.RepoURL, err)
	}
	selectedTag, selectedCommit, err := r.selectTagAndCommitID(repo, sub)
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
) (string, string, error) {
	if sub.CommitSelectionStrategy == kargoapi.CommitSelectionStrategyNewestFromBranch {
		// In this case, there is nothing to do except return the commit ID at the
		// head of the branch.
		commit, err := r.getLastCommitIDFn(repo)
		if err != nil {
			return "", "", fmt.Errorf(
				"error determining commit ID at head of branch %q in git repo %q: %w",
				sub.Branch,
				sub.RepoURL,
				err,
			)
		}
		return "", commit, nil
	}

	tags, err := r.listTagsFn(repo) // These are ordered newest to oldest
	if err != nil {
		return "", "", fmt.Errorf("error listing tags from git repo %q: %w", sub.RepoURL, err)
	}

	// Narrow down the list of tags to those that are allowed and not ignored
	allowRegex, err := regexp.Compile(sub.AllowTags)
	if err != nil {
		return "", "", fmt.Errorf("error compiling regular expression %q: %w", sub.AllowTags, err)
	}
	filteredTags := make([]string, 0, len(tags))
	for _, tagName := range tags {
		if allows(tagName, allowRegex) && !ignores(tagName, sub.IgnoreTags) {
			filteredTags = append(filteredTags, tagName)
		}
	}
	if len(filteredTags) == 0 {
		return "", "", fmt.Errorf("found no applicable tags in repo %q", sub.RepoURL)
	}

	var selectedTag string
	switch sub.CommitSelectionStrategy {
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
	if selectedTag == "" {
		return "", "", fmt.Errorf("found no applicable tags in repo %q", sub.RepoURL)
	}

	// Checkout the selected tag and return the commit ID
	if err = r.checkoutTagFn(repo, selectedTag); err != nil {
		return "", "", fmt.Errorf(
			"error checking out tag %q from git repo %q: %w",
			selectedTag,
			sub.RepoURL,
			err,
		)
	}
	commit, err := r.getLastCommitIDFn(repo)
	if err != nil {
		return "", "", fmt.Errorf(
			"error determining commit ID of tag %q in git repo %q: %w",
			selectedTag,
			sub.RepoURL,
			err,
		)

	}
	return selectedTag, commit, nil
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
