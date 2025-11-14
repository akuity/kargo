package commit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/logging"
)

const branchPrefix = "refs/heads/"

func init() {
	defaultSelectorRegistry.MustRegister(
		selectorRegistration{
			Predicate: func(_ context.Context, sub kargoapi.GitSubscription) (bool, error) {
				return sub.CommitSelectionStrategy == kargoapi.CommitSelectionStrategyNewestFromBranch ||
					sub.CommitSelectionStrategy == "", nil
			},
			Value: newNewestFromBranchSelector,
		},
	)
}

// newestFromBranchSelector implements the Selector interface for
// kargoapi.CommitSelectionStrategyNewestFromBranch.
type newestFromBranchSelector struct {
	*baseSelector
	branch string

	selectCommitsFn func(git.Repo) ([]git.CommitMetadata, error)
	listCommitsFn   func(
		repo git.Repo,
		limit uint,
		skip uint,
	) ([]git.CommitMetadata, error)
	getDiffPathsForCommitIDFn func(
		repo git.Repo,
		commitID string,
	) ([]string, error)
}

func newNewestFromBranchSelector(
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (Selector, error) {
	base, err := newBaseSelector(sub, creds)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}
	s := &newestFromBranchSelector{
		baseSelector: base,
		branch:       sub.Branch,
	}
	s.selectCommitsFn = s.selectCommits
	s.listCommitsFn = s.listCommits
	s.getDiffPathsForCommitIDFn = s.getDiffPathsForCommitID
	return s, nil
}

// MatchesRef implements Selector.
func (n *newestFromBranchSelector) MatchesRef(ref string) bool {
	if !strings.HasPrefix(ref, branchPrefix) {
		return false // Doesn't look like a branch ref
	}
	branch := strings.TrimPrefix(ref, "refs/heads/")
	if n.branch == "" {
		// Note: We do not require branch to be specified in the subscription.
		// When it is not specified, the subscription is implicitly for the
		// repository's default branch. We don't really know what the repository's
		// default branch is, so we make two imperfect assumptions here:
		//   1. The default branch is either "main" or "master".
		//   2. The repository does not have BOTH a main and master branch.
		return branch == "main" || branch == "master"
	}
	return n.branch == branch
}

// Select implements the Selector interface.
func (n *newestFromBranchSelector) Select(ctx context.Context) (
	[]kargoapi.DiscoveredCommit,
	error,
) {
	loggerCtx := append(
		n.getLoggerContext(),
		"selectionStrategy", kargoapi.CommitSelectionStrategyNewestFromBranch,
		"branch", n.branch,
	)
	logger := logging.LoggerFromContext(ctx).WithValues(loggerCtx...)
	ctx = logging.ContextWithLogger(ctx, logger)

	logger.Debug("cloning repository")
	repo, err := n.gitCloneFn(
		n.repoURL,
		&git.ClientOptions{
			Credentials:           n.creds,
			InsecureSkipTLSVerify: n.insecureSkipTLSVerify,
		},
		&git.CloneOptions{
			Branch:       n.branch,
			SingleBranch: true,
			Filter:       git.FilterBlobless,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error cloning git repo %q: %w", n.repoURL, err)
	}
	defer func() {
		_ = repo.Close()
	}()

	commits, err := n.selectCommitsFn(repo)
	if err != nil {
		return nil,
			fmt.Errorf("error selecting relevant commits from branch: %w", err)
	}

	return n.commitsToAPICommits(ctx, commits), nil
}

func (n *newestFromBranchSelector) selectCommits(
	repo git.Repo,
) ([]git.CommitMetadata, error) {
	selectedCommits := make([]git.CommitMetadata, 0, n.discoveryLimit)
	for skip, batch := uint(0), uint(n.discoveryLimit); ; skip, batch = skip+batch, min(batch*2, 1000) { // nolint: gosec
		commits, err := n.listCommitsFn(repo, batch, skip) // nolint: gosec
		if err != nil {
			return nil,
				fmt.Errorf("error listing commits from git repo %q: %w", n.repoURL, err)
		}

		if len(commits) == 0 {
			break
		}

		// If no filters are specified, return the first commits up to the limit.
		if n.includePaths == nil && n.excludePaths == nil && n.filterExpression == nil {
			return trimSlice(commits, n.discoveryLimit), nil
		}

		for _, commit := range commits {
			// Filter commits based on expressions.
			include, err := n.evaluateCommitExpression(commit)
			if err != nil {
				return nil, fmt.Errorf("error evaluating filter expression: %w", err)
			}
			if !include {
				continue
			}

			// If include or exclude path selectors are specified, filter the commits.
			if n.includePaths != nil || n.excludePaths != nil {
				diffPaths, err := n.getDiffPathsForCommitIDFn(repo, commit.ID)
				if err != nil {
					return nil, fmt.Errorf(
						"error getting diff paths for commit %q in git repo %q: %w",
						commit.ID,
						n.repoURL,
						err,
					)
				}
				if !matchesPathsFilters(n.includePaths, n.excludePaths, diffPaths) {
					continue
				}
			}

			// If we reach this point, the commit got past all the filters.
			selectedCommits = append(selectedCommits, commit)

			if len(selectedCommits) >= n.discoveryLimit {
				return trimSlice(selectedCommits, n.discoveryLimit), nil
			}
		}
	}

	return trimSlice(selectedCommits, n.discoveryLimit), nil
}

func (n *newestFromBranchSelector) listCommits(
	repo git.Repo,
	limit uint,
	skip uint,
) ([]git.CommitMetadata, error) {
	return repo.ListCommits(limit, skip) // nolint: gosec
}

func (n *newestFromBranchSelector) getDiffPathsForCommitID(
	repo git.Repo,
	commitID string,
) ([]string, error) {
	return repo.GetDiffPathsForCommitID(commitID)
}

// evaluateCommitExpression evaluates the given commit expression against
// the given commit metadata. The commit metadata is passed as the environment
// for the expression evaluation. It returns true if the expression evaluates to
// true, and false otherwise. If the expression is not a boolean, it is
// converted to a boolean using strconv.ParseBool. If the conversion fails,
// an error is returned.
func (n *newestFromBranchSelector) evaluateCommitExpression(
	commit git.CommitMetadata,
) (bool, error) {
	if n.filterExpression == nil {
		return true, nil
	}

	env := map[string]any{
		"id":         commit.ID,
		"commitDate": commit.CommitDate,
		"author":     commit.Author,
		"committer":  commit.Committer,
		"subject":    commit.Subject,
	}

	result, err := expr.Run(n.filterExpression, env)
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

func (n *newestFromBranchSelector) commitsToAPICommits(
	ctx context.Context,
	commits []git.CommitMetadata,
) []kargoapi.DiscoveredCommit {
	logger := logging.LoggerFromContext(ctx)
	apiCommits := make([]kargoapi.DiscoveredCommit, 0, len(commits))
	for _, meta := range commits {
		apiCommits = append(apiCommits, kargoapi.DiscoveredCommit{
			ID:     meta.ID,
			Branch: n.branch,
			// A decent subject length for a commit message is 50 characters
			// (based on the 50/72 rule). We are nice people, and allow a
			// bit more. But not an excessive amount, to minimize the risk of
			// exceeding the maximum size of the object in the API server.
			Subject:     shortenString(meta.Subject, 80),
			Author:      meta.Author,
			Committer:   meta.Committer,
			CreatorDate: &metav1.Time{Time: meta.CommitDate},
		})
		logger.Trace(
			"discovered commit from branch",
			"commit", meta.ID,
			"creatorDate", meta.CommitDate.Format(time.RFC3339),
		)
	}
	logger.Debug("discovered commits", "count", len(commits))
	return apiCommits
}
