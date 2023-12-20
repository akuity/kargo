package gitprovider

import (
	"context"
)

// GitProviderService is an abstracted interface for a git providers (GitHub, GitLab, BitBucket)
// when interacting against a single git repository (e.g. managing pull requests).
type GitProviderService interface { // nolint: revive
	// WithAuthToken sets an authorization token to the client
	WithAuthToken(token string) GitProviderService

	// CreatePullRequest creates a pull request
	CreatePullRequest(ctx context.Context, repoURL string, opts CreatePullRequestOpts) (*PullRequest, error)

	// Get gets an existing pull request by ID
	GetPullRequest(ctx context.Context, repoURL string, number int64) (*PullRequest, error)

	// ListPullRequests lists pull requests by the given options
	ListPullRequests(ctx context.Context, repoURL string, opts ListPullRequestOpts) ([]*PullRequest, error)

	// IsPullRequestMerged returns whether or not the pull request was merged
	IsPullRequestMerged(ctx context.Context, repoURL string, number int64) (bool, error)
}

type CreatePullRequestOpts struct {
	Head        string
	Base        string
	Title       string
	Description string
}

type ListPullRequestOpts struct {
	// State is the pull request state (one of: Open, Closed). Defaults to Open
	State PullRequestState
	Head  string
	Base  string
}

type PullRequestState string

const (
	PullRequestStateOpen   PullRequestState = "Open"
	PullRequestStateClosed PullRequestState = "Closed"
)

type PullRequest struct {
	// Number is the numeric pull request number (not an ID)
	// Pull requests numbers are unique only within a repository
	Number int64 `json:"id"`
	// URL is the url to the pull request
	URL string `json:"url"`
	// State is the pull request state (one of: Open, Closed)
	State PullRequestState `json:"state"`
	// MergeCommitSHA is the SHA of the merge commit
	MergeCommitSHA string `json:"mergeCommitSHA"`
	// Object is the underlying object from the provider
	Object any `json:"-"`
}

func (pr *PullRequest) IsOpen() bool {
	return pr.State == PullRequestStateOpen
}
