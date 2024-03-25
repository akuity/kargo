package promotion

import (
	"context"
	"fmt"
	"strconv"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/akuity/kargo/internal/gitprovider/github"
)

func pullRequestBranchName(project, stage string) string {
	return fmt.Sprintf("kargo/%s/%s/promotion", project, stage)
}

// preparePullRequestBranch prepares a branch to be used as the pull request branch for
// merging into the base branch. If the PR branch already exists, but not in a state that
// we like (i.e. not a descendant of base), recreate it.
func preparePullRequestBranch(repo git.Repo, prBranch string, base string) error {
	origBranch := repo.CurrentBranch()
	baseBranchExists, err := repo.RemoteBranchExists(base)
	if err != nil {
		return err
	}
	if !baseBranchExists {
		// Base branch doesn't exist. Create it!
		if err = repo.CreateOrphanedBranch(base); err != nil {
			return err
		}
		if err = repo.Commit(
			"Initial commit",
			&git.CommitOptions{
				AllowEmpty: true,
			},
		); err != nil {
			return err
		}
		if err = repo.Push(false); err != nil {
			return err
		}
	} else if err = repo.Checkout(base); err != nil {
		return err
	}
	prBranchExists, err := repo.RemoteBranchExists(prBranch)
	if err != nil {
		return err
	}
	if !prBranchExists {
		// PR branch doesn't exist
		if err := repo.CreateChildBranch(prBranch); err != nil {
			return err
		}
		if err := repo.Push(false); err != nil {
			return err
		}
	} else {
		// PR branch exists, ensure writeBranch is an ancestor.
		// otherwise PRs cannot be created.
		if err := repo.Checkout(prBranch); err != nil {
			return err
		}
		isAncestor, err := repo.IsAncestor(base, prBranch)
		if err != nil {
			return err
		}
		if !isAncestor {
			// Branch exists, but is not an ancestor of writeBranch, recreate it
			if err = repo.Checkout(base); err != nil {
				return err
			}
			if err = repo.DeleteBranch(prBranch); err != nil {
				return err
			}
			if err = repo.CreateChildBranch(prBranch); err != nil {
				return err
			}
			if err = repo.Push(true); err != nil {
				return err
			}
		}
	}
	// Return to original branch
	return repo.Checkout(origBranch)
}

// newGitProvider returns the appropriate git provider either if it was explicitly specified,
// or if we can infer it from the repo URL.
func newGitProvider(
	url string,
	pullRequest *kargoapi.PullRequestPromotionMechanism,
	creds *git.RepoCredentials,
) (gitprovider.GitProviderService, error) {
	var gpClient gitprovider.GitProviderService
	var err error
	if pullRequest.GitHub != nil {
		gpClient, err = gitprovider.NewGitProviderServiceFromName(github.GitProviderServiceName)
	} else {
		gpClient, err = gitprovider.NewGitProviderServiceFromURL(url)
	}
	if err != nil {
		return nil, err
	}
	if creds != nil {
		gpClient = gpClient.WithAuthToken(creds.Password)
	}
	return gpClient, nil
}

// reconcilePullRequest creates and monitors a pull request for the promotion,
// then returns a PromotionStatus reflecting current status adding metadata
// it tracks (i.e. PR url).
func reconcilePullRequest(
	ctx context.Context,
	status kargoapi.PromotionStatus,
	repo git.Repo,
	gpClient gitprovider.GitProviderService,
	prBranch string,
	writeBranch string,
) (string, *kargoapi.PromotionStatus, error) {
	newStatus := status.DeepCopy()
	var mergeCommitSHA string

	prNumber := getPullRequestNumberFromMetadata(status.Metadata, repo.URL())
	if prNumber == -1 {
		needsPR, err := repo.RefsHaveDiffs(prBranch, writeBranch)
		if err != nil {
			return "", nil, err
		}
		if needsPR {
			title, err := repo.CommitMessage(prBranch)
			if err != nil {
				return "", nil, err
			}
			createOpts := gitprovider.CreatePullRequestOpts{
				Head:  prBranch,
				Base:  writeBranch,
				Title: title,
			}
			pr, err := gpClient.CreatePullRequest(ctx, repo.URL(), createOpts)
			if err != nil {
				// Error might be "A pull request already exists" for same branches.
				// Check if that is the case, and reuse the existing PR if it is
				prs, listErr := gpClient.ListPullRequests(ctx, repo.URL(), gitprovider.ListPullRequestOpts{
					Head: prBranch,
					Base: writeBranch,
				})
				if listErr != nil || len(prs) != 1 {
					return "", nil, err
				}
				// If we get here, we found an existing open PR for the same branches
				pr = prs[0]
			}
			newStatus.Phase = kargoapi.PromotionPhaseRunning
			newStatus.Metadata = setPullRequestMetadata(newStatus.Metadata, repo.URL(), pr.Number, pr.URL)
		} else {
			newStatus.Phase = kargoapi.PromotionPhaseSucceeded
			newStatus.Message = "No changes to promote"
		}
	} else {
		// check if existing PR is closed/merged and update promo status to either
		// Succeeded or Failed depending if PR was merged
		pr, err := gpClient.GetPullRequest(ctx, repo.URL(), prNumber)
		if err != nil {
			return "", nil, err
		}
		if !pr.IsOpen() {
			merged, err := gpClient.IsPullRequestMerged(ctx, repo.URL(), prNumber)
			if err != nil {
				return "", nil, err
			}
			if merged {
				newStatus.Phase = kargoapi.PromotionPhaseSucceeded
				newStatus.Message = "Pull request was merged"
				if pr.MergeCommitSHA == "" {
					return "", nil, fmt.Errorf("merge commit SHA is empty")
				}
				mergeCommitSHA = pr.MergeCommitSHA
			} else {
				newStatus.Phase = kargoapi.PromotionPhaseFailed
				newStatus.Message = "Pull request was closed without being merged"
			}
		}
	}

	return mergeCommitSHA, newStatus, nil
}

// pullRequestMetadataKey returns the key used to store the pull request number in the metadata map.
func pullRequestMetadataKey(repoURL string) string {
	return fmt.Sprintf("pr:%s", repoURL)
}

// setPullRequestMetadata sets pull request bookkeeping information to the metadata map.
func setPullRequestMetadata(metadata map[string]string, repoURL string, number int64, url string) map[string]string {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata[pullRequestMetadataKey(repoURL)] = strconv.FormatInt(number, 10)
	// we only set url for UI purposes so there is no helper function for key
	metadata[fmt.Sprintf("pr-url:%s", repoURL)] = url
	return metadata
}

// getPullRequestNumberFromMetadata returns the pull request number and URL from the metadata map.
// If no pull request number is found, -1 is returned.
func getPullRequestNumberFromMetadata(metadata map[string]string, repoURL string) int64 {
	if metadata == nil {
		return -1
	}
	prNumStr := metadata[pullRequestMetadataKey(repoURL)]
	if prNumStr == "" {
		return -1
	}
	intVal, err := strconv.ParseInt(prNumStr, 10, 0)
	if err != nil {
		return -1
	}
	return intVal
}
