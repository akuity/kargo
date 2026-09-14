package utils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v76/github"
)

// CreateRemoteBranch creates branch in the GitHub repository, pointing it at the
// current head of fromBranch, using the go-github API. Branch ref management is
// not covered by the gitprovider abstraction, and the e2e suites target a GitHub
// fork, so this uses go-github directly.
func CreateRemoteBranch(ctx context.Context, repoURL, token, branch, fromBranch string) error {
	owner, repo, err := gitHubOwnerRepo(repoURL)
	if err != nil {
		return err
	}

	client := github.NewClient(nil).WithAuthToken(token)

	base, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+fromBranch)
	if err != nil {
		return fmt.Errorf("error getting ref for branch %q: %w", fromBranch, err)
	}
	if base.GetObject().GetSHA() == "" {
		return fmt.Errorf("no sha found for branch %q", fromBranch)
	}

	if _, _, err := client.Git.CreateRef(ctx, owner, repo, github.CreateRef{
		Ref: "refs/heads/" + branch,
		SHA: base.GetObject().GetSHA(),
	}); err != nil {
		return fmt.Errorf("error creating branch %q: %w", branch, err)
	}
	return nil
}

// RemoteBranchSHA returns the commit SHA at the head of the given branch in the
// GitHub repository.
func RemoteBranchSHA(ctx context.Context, repoURL, token, branch string) (string, error) {
	owner, repo, err := gitHubOwnerRepo(repoURL)
	if err != nil {
		return "", err
	}
	client := github.NewClient(nil).WithAuthToken(token)
	ref, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("error getting ref for branch %q: %w", branch, err)
	}
	return ref.GetObject().GetSHA(), nil
}

// EnsureLabel creates a label in the GitHub repository if it does not already
// exist. git-open-pr applies labels via the issues API, which requires the
// labels to exist beforehand.
func EnsureLabel(ctx context.Context, repoURL, token, name string) error {
	owner, repo, err := gitHubOwnerRepo(repoURL)
	if err != nil {
		return err
	}
	client := github.NewClient(nil).WithAuthToken(token)
	_, resp, err := client.Issues.CreateLabel(ctx, owner, repo, &github.Label{
		Name:  github.Ptr(name),
		Color: github.Ptr("ededed"),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
			// The label already exists.
			return nil
		}
		return fmt.Errorf("error creating label %q: %w", name, err)
	}
	return nil
}

// PullRequestInfo is the subset of a GitHub pull request used for assertions.
type PullRequestInfo struct {
	Title  string
	Body   string
	Labels []string
}

// GetPullRequest fetches the pull request with the given number.
func GetPullRequest(ctx context.Context, repoURL, token string, number int) (*PullRequestInfo, error) {
	owner, repo, err := gitHubOwnerRepo(repoURL)
	if err != nil {
		return nil, err
	}
	client := github.NewClient(nil).WithAuthToken(token)
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request %d: %w", number, err)
	}
	labels := make([]string, 0, len(pr.Labels))
	for _, label := range pr.Labels {
		labels = append(labels, label.GetName())
	}
	return &PullRequestInfo{
		Title:  pr.GetTitle(),
		Body:   pr.GetBody(),
		Labels: labels,
	}, nil
}
