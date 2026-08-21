package utils

import (
	"context"
	"fmt"

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
