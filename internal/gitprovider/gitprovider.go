package gitprovider

import (
	"context"
)

// GitProviderOptions contains the options for a GitProvider.
type GitProviderOptions struct { // nolint: revive
	// Name specifies which Git provider to use when that information cannot be
	// inferred from the repository URL.
	Name string
	// Token is the access token used to authenticate against the Git provider's
	// API.
	Token string
	// InsecureSkipTLSVerify specifies whether certificate verification errors
	// should be ignored when connecting to the Git provider's API.
	InsecureSkipTLSVerify bool
}

// GitProviderService is an abstracted interface for a git providers (GitHub, GitLab, BitBucket)
// when interacting against a single git repository (e.g. managing pull requests).
type GitProviderService interface { // nolint: revive
	// CreatePullRequest creates a pull request
	CreatePullRequest(ctx context.Context, opts CreatePullRequestOpts) (*PullRequest, error)

	// Get gets an existing pull request by ID
	GetPullRequest(ctx context.Context, number int64) (*PullRequest, error)

	// ListPullRequests lists pull requests by the given options
	ListPullRequests(ctx context.Context, opts ListPullRequestOpts) ([]*PullRequest, error)

	// IsPullRequestMerged returns whether or not the pull request was merged
	IsPullRequestMerged(ctx context.Context, number int64) (bool, error)
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
	// HeadSHA is the SHA of the head commit
	HeadSHA string `json:"headSHA"`
}

func (pr *PullRequest) IsOpen() bool {
	return pr.State == PullRequestStateOpen
}

type FakeGitProviderService struct {
	CreatePullRequestFn func(
		context.Context,
		CreatePullRequestOpts,
	) (*PullRequest, error)
	GetPullRequestFn   func(context.Context, int64) (*PullRequest, error)
	ListPullRequestsFn func(
		context.Context,
		ListPullRequestOpts,
	) ([]*PullRequest, error)
	IsPullRequestMergedFn func(context.Context, int64) (bool, error)
}

func (f *FakeGitProviderService) CreatePullRequest(
	ctx context.Context,
	opts CreatePullRequestOpts,
) (*PullRequest, error) {
	return f.CreatePullRequestFn(ctx, opts)
}

func (f *FakeGitProviderService) GetPullRequest(
	ctx context.Context,
	number int64,
) (*PullRequest, error) {
	return f.GetPullRequestFn(ctx, number)
}

func (f *FakeGitProviderService) ListPullRequests(
	ctx context.Context,
	opts ListPullRequestOpts,
) ([]*PullRequest, error) {
	return f.ListPullRequestsFn(ctx, opts)
}

func (f *FakeGitProviderService) IsPullRequestMerged(
	ctx context.Context,
	number int64,
) (bool, error) {
	return f.IsPullRequestMergedFn(ctx, number)
}
