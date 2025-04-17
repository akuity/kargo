package gitprovider

import (
	"context"
	"time"
)

// PullRequestState represents the state of a pull request. e.g. Closed, Open,
// etc.
type PullRequestState string

const (
	// PullRequestStateAny represents all pull requests, regardless of state.
	PullRequestStateAny PullRequestState = "Any"
	// PullRequestStateClosed represents pull requests that are (logically)
	// closed. Depending on the underlying provider, this may encompass pull
	// requests in other states such as "merged" or "declined".
	PullRequestStateClosed PullRequestState = "Closed"
	// PullRequestStateOpen represents pull requests that are (logically) open.
	// Depending on the underlying provider, this may encompass pull requests in
	// other states such as "draft" or "ready for review".
	PullRequestStateOpen PullRequestState = "Open"
)

// Options encapsulates options used in instantiating any implementation
// of Interface.
type Options struct {
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

// Interface is an abstracted interface for interacting with a single repository
// hosted by some a Git hosting provider (e.g. GitHub, GitLab, BitBucket,
// etc.).
type Interface interface {
	// CreatePullRequest creates a pull request.
	CreatePullRequest(context.Context, *CreatePullRequestOpts) (*PullRequest, error)

	// Get gets an existing pull request by ID
	GetPullRequest(context.Context, int64) (*PullRequest, error)

	// ListPullRequests lists pull requests by the given options. Implementations
	// have no obligation to sort the results in any particular order, due mainly
	// to differences in the underlying provider APIs. It is the responsibility of
	// the caller to sort the results as needed.
	ListPullRequests(context.Context, *ListPullRequestOptions) ([]PullRequest, error)

	// GetCommitURL get a commit by the repository URL and commit SHA.
	GetCommitURL(string, string) (string, error)
}

// CreatePullRequestOpts encapsulates the options used when creating a pull
// request.
type CreatePullRequestOpts struct {
	// Title is the title of the pull request.
	Title string
	// Description is the body of the pull request.
	Description string
	// Head is the name of the source branch.
	Head string
	// Base is the name of the target branch.
	Base string
	// Labels is an array of strings that should be added as labels to the pull request.
	Labels []string
}

// ListPullRequestOptions encapsulates the options used when listing pull
// requests.
type ListPullRequestOptions struct {
	// State is the pull request state (one of: Any, Closed, or Open).
	State PullRequestState
	// HeadBranch is the name of the source branch. A non-empty value will limit
	// results to pull requests with the given source branch.
	HeadBranch string
	// HeadCommit is the SHA of the commit at the head of the source branch. A
	// non-empty value will limit results to pull requests with the given head
	// commit.
	HeadCommit string
	// BaseBranch is the name of the target branch. A non-empty value will limit
	// results to pull requests with the given target branch.
	BaseBranch string
}

// PullRequest is an abstracted representation of a Git hosting provider's pull
// request object (or equivalent; e.g. a GitLab merge request).
type PullRequest struct {
	// Number is the pull request number, which is unique only within a single
	// repository.
	Number int64 `json:"id"`
	// URL is the URL to the pull request.
	URL string `json:"url"`
	// Open is true if the pull request is logically open. Depending on the
	// underlying Git hosting provider, this may encompass pull requests in other
	// states such as "draft" or "ready for review".
	Open bool `json:"open"`
	// Merged is true if the pull request was merged.
	Merged bool `json:"merged"`
	// MergeCommitSHA is the SHA of the merge commit.
	MergeCommitSHA string `json:"mergeCommitSHA"`
	// Object is the underlying object from the Git hosting provider.
	Object any `json:"-"`
	// HeadSHA is the SHA of the commit at the head of the source branch.
	HeadSHA string `json:"headSHA"`
	// CreatedAt is the time the pull request was created.
	CreatedAt *time.Time `json:"createdAt"`
}

// Fake is a fake implementation of the provider Interface used to facilitate
// testing.
type Fake struct {
	// CreatePullRequestFn defines the functionality of the CreatePullRequest
	// method.
	CreatePullRequestFn func(
		context.Context,
		*CreatePullRequestOpts,
	) (*PullRequest, error)
	// GetPullRequestFn defines the functionality of the GetPullRequest method.
	GetPullRequestFn func(context.Context, int64) (*PullRequest, error)
	// ListPullRequestsFn defines the functionality of the ListPullRequests
	// method.
	ListPullRequestsFn func(
		context.Context,
		*ListPullRequestOptions,
	) ([]PullRequest, error)
	// GetCommitURLFn defines the functionality of the GetCommitURL method.
	GetCommitURLFn func(string, string) (string, error)
}

// CreatePullRequest implements gitprovider.Interface.
func (f *Fake) CreatePullRequest(
	ctx context.Context,
	opts *CreatePullRequestOpts,
) (*PullRequest, error) {
	return f.CreatePullRequestFn(ctx, opts)
}

// GetPullRequest implements gitprovider.Interface.
func (f *Fake) GetPullRequest(
	ctx context.Context,
	number int64,
) (*PullRequest, error) {
	return f.GetPullRequestFn(ctx, number)
}

// ListPullRequests implements gitprovider.Interface.
func (f *Fake) ListPullRequests(
	ctx context.Context,
	opts *ListPullRequestOptions,
) ([]PullRequest, error) {
	return f.ListPullRequestsFn(ctx, opts)
}

// GetCommitURL implements gitprovider.Interface.
func (f *Fake) GetCommitURL(
	repoURL string,
	sha string,
) (string, error) {
	return f.GetCommitURLFn(repoURL, sha)
}
