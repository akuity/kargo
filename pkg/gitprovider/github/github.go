package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v76/github"
	"k8s.io/utils/ptr"

	ghutil "github.com/akuity/kargo/pkg/github"
	"github.com/akuity/kargo/pkg/gitprovider"
)

const ProviderName = "github"

// GitHub pull request states
const (
	prStateAll    = "all"
	prStateClosed = "closed"
	prStateOpen   = "open"
)

// GitHub pull request mergeable states. GitHub computes mergeability
// asynchronously and reports the outcome via the PR's mergeable_state field.
// These values are not formally documented but are stable in practice.
const (
	// mergeableStateClean indicates the PR is mergeable and all required checks
	// have passed.
	mergeableStateClean = "clean"
	// mergeableStateUnstable indicates the PR is mergeable, but some
	// non-required checks are failing.
	mergeableStateUnstable = "unstable"
	// mergeableStateHasHooks indicates the PR is mergeable and pre-receive hooks
	// are configured.
	mergeableStateHasHooks = "has_hooks"
	// mergeableStateBlocked indicates the merge is blocked by required reviews or
	// checks that have not yet been satisfied.
	mergeableStateBlocked = "blocked"
	// mergeableStateBehind indicates the head branch is behind the base branch,
	// typically because the base branch moved forward.
	mergeableStateBehind = "behind"
	// mergeableStateDraft indicates the PR is a draft and cannot be merged.
	mergeableStateDraft = "draft"
	// mergeableStateDirty indicates the PR has merge conflicts.
	mergeableStateDirty = "dirty"
	// mergeableStateUnknown indicates GitHub has not finished computing
	// mergeability.
	mergeableStateUnknown = "unknown"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// We assume that any hostname with the word "github" in it can use this
		// provider. We also explicitly support 'ghe.com' (GitHub Enterprise Cloud).
		// NOTE: We will miss cases where the host is GitHub Enterprise
		// but doesn't incorporate the word "github" or "ghe.com" (e.g. 'git.mycompany.com').
		return strings.Contains(u.Host, ProviderName) || strings.HasSuffix(u.Host, ".ghe.com")
	},
	NewProvider: func(
		repoURL string,
		opts *gitprovider.Options,
	) (gitprovider.Interface, error) {
		return NewProvider(repoURL, opts)
	},
}

func init() {
	gitprovider.Register(ProviderName, registration)
}

type githubClient interface {
	CreatePullRequest(
		ctx context.Context,
		owner string,
		repo string,
		pull *github.NewPullRequest,
	) (*github.PullRequest, *github.Response, error)

	ListPullRequests(
		ctx context.Context,
		owner string,
		repo string,
		opts *github.PullRequestListOptions,
	) ([]*github.PullRequest, *github.Response, error)

	GetPullRequests(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) (*github.PullRequest, *github.Response, error)

	MergePullRequest(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		commitMessage string,
		options *github.PullRequestOptions,
	) (*github.PullRequestMergeResult, *github.Response, error)

	AddLabelsToIssue(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		labels []string,
	) ([]*github.Label, *github.Response, error)
}

// provider is a GitHub implementation of gitprovider.Interface.
type provider struct { // nolint: revive
	owner  string
	repo   string
	client githubClient
}

// NewProvider returns a GitHub-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}

	_, _, owner, repo, err := ghutil.ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	client, err := ghutil.NewClient(repoURL, &ghutil.ClientOptions{
		Token:                 opts.Token,
		InsecureSkipTLSVerify: opts.InsecureSkipTLSVerify,
	})
	if err != nil {
		return nil, err
	}

	return &provider{
		owner:  owner,
		repo:   repo,
		client: &githubClientWrapper{client},
	}, nil
}

type githubClientWrapper struct {
	client *github.Client
}

func (g githubClientWrapper) CreatePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	pull *github.NewPullRequest,
) (*github.PullRequest, *github.Response, error) {
	return g.client.PullRequests.Create(ctx, owner, repo, pull)
}

func (g githubClientWrapper) ListPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	return g.client.PullRequests.List(ctx, owner, repo, opts)
}

func (g githubClientWrapper) GetPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*github.PullRequest, *github.Response, error) {
	return g.client.PullRequests.Get(ctx, owner, repo, number)
}

func (g githubClientWrapper) MergePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	commitMessage string,
	options *github.PullRequestOptions,
) (*github.PullRequestMergeResult, *github.Response, error) {
	return g.client.PullRequests.Merge(ctx, owner, repo, number, commitMessage, options)
}

func (g githubClientWrapper) AddLabelsToIssue(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	labels []string,
) ([]*github.Label, *github.Response, error) {
	return g.client.Issues.AddLabelsToIssue(ctx, owner, repo, number, labels)
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	ctx context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.CreatePullRequestOpts{}
	}
	ghPR, _, err := p.client.CreatePullRequest(ctx,
		p.owner,
		p.repo,
		&github.NewPullRequest{
			Title:               &opts.Title,
			Head:                &opts.Head,
			Base:                &opts.Base,
			Body:                &opts.Description,
			MaintainerCanModify: github.Ptr(false),
		},
	)
	if err != nil {
		return nil, err
	}
	if ghPR == nil {
		return nil, fmt.Errorf("unexpected nil pull request")
	}
	pr := convertGithubPR(*ghPR)
	if len(opts.Labels) > 0 {
		_, _, err = p.client.AddLabelsToIssue(ctx,
			p.owner,
			p.repo,
			int(pr.Number),
			opts.Labels,
		)
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	ghPR, _, err := p.client.GetPullRequests(ctx, p.owner, p.repo, int(id))
	if err != nil {
		return nil, err
	}
	if ghPR == nil {
		return nil, fmt.Errorf("unexpected nil pull request")
	}
	pr := convertGithubPR(*ghPR)
	return &pr, nil
}

// ListPullRequests implements gitprovider.Interface.
func (p *provider) ListPullRequests(
	ctx context.Context,
	opts *gitprovider.ListPullRequestOptions,
) ([]gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.ListPullRequestOptions{}
	}
	if opts.State == "" {
		opts.State = gitprovider.PullRequestStateOpen
	}
	listOpts := github.PullRequestListOptions{
		Head: opts.HeadBranch,
		Base: opts.BaseBranch,
		ListOptions: github.ListOptions{
			PerPage: 100, // Max
		},
	}
	switch opts.State {
	case gitprovider.PullRequestStateAny:
		listOpts.State = prStateAll
	case gitprovider.PullRequestStateClosed:
		listOpts.State = prStateClosed
	case gitprovider.PullRequestStateOpen:
		listOpts.State = prStateOpen
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}
	var prs []gitprovider.PullRequest
	for {
		ghPRs, res, err := p.client.ListPullRequests(ctx, p.owner, p.repo, &listOpts)
		if err != nil {
			return nil, err
		}
		for _, ghPR := range ghPRs {
			if opts.HeadCommit == "" || ptr.Deref(ghPR.Head.SHA, "") == opts.HeadCommit {
				prs = append(prs, convertGithubPR(*ghPR))
			}
		}
		if res == nil || res.NextPage == 0 {
			break
		}
		listOpts.Page = res.NextPage
	}

	return prs, nil
}

// MergePullRequest implements gitprovider.Interface.
func (p *provider) MergePullRequest(
	ctx context.Context,
	id int64,
	opts *gitprovider.MergePullRequestOpts,
) (*gitprovider.PullRequest, bool, error) {
	if opts == nil {
		opts = &gitprovider.MergePullRequestOpts{}
	}

	ghPR, _, err := p.client.GetPullRequests(ctx, p.owner, p.repo, int(id))
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if ghPR == nil {
		return nil, false, fmt.Errorf("pull request %d not found", id)
	}

	if ghPR.MergedAt != nil {
		pr := convertGithubPR(*ghPR)
		return &pr, true, nil
	}
	if ptr.Deref(ghPR.State, prStateClosed) != prStateOpen {
		return nil, false, fmt.Errorf("pull request %d is closed but not merged", id)
	}

	// Decide whether to attempt the merge based on GitHub's mergeable_state. This
	// is richer than the Mergeable boolean: it lets us distinguish a permanent
	// conflict (which will never resolve on its own) from transient conditions
	// such as pending checks, a base branch that has moved forward, or
	// mergeability that GitHub is still computing -- all of which a caller should
	// retry rather than fail on.
	switch ghPR.GetMergeableState() {
	case mergeableStateClean, mergeableStateUnstable, mergeableStateHasHooks:
		// Mergeable now; fall through to the merge attempt below.
	case mergeableStateDirty:
		// A genuine merge conflict will not clear without human intervention.
		return nil, false, fmt.Errorf("pull request %d has conflicts and cannot be merged", id)
	case mergeableStateBlocked, mergeableStateBehind, mergeableStateDraft, mergeableStateUnknown:
		// Not ready to merge yet, but the condition may clear; signal "not ready"
		// so the caller can retry on a subsequent reconciliation.
		return nil, false, nil
	default:
		// mergeable_state is unset or unrecognized; fall back to the Mergeable
		// boolean and the draft flag.
		if ptr.Deref(ghPR.Draft, false) || !ptr.Deref(ghPR.Mergeable, false) {
			return nil, false, nil
		}
	}

	// Merge the PR
	mergeResult, _, err := p.client.MergePullRequest(
		ctx,
		p.owner,
		p.repo,
		int(id),
		"", // Use default commit message.
		&github.PullRequestOptions{MergeMethod: opts.MergeMethod},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}
	if mergeResult == nil {
		return nil, false, fmt.Errorf("unexpected nil merge result")
	}
	if !ptr.Deref(mergeResult.Merged, false) {
		return nil, false,
			fmt.Errorf("merge rejected for pull request %d", id)
	}

	updatedPR, _, err := p.client.GetPullRequests(ctx, p.owner, p.repo, int(id))
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d after merge: %w", id, err)
	}
	if updatedPR == nil {
		return nil, false, fmt.Errorf("unexpected nil pull request after merge")
	}

	pr := convertGithubPR(*updatedPR)

	return &pr, true, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(
	repoURL string,
	sha string,
) (string, error) {
	_, host, owner, repo, err := ghutil.ParseRepoURL(repoURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}
	return fmt.Sprintf("https://%s/%s/%s/commit/%s", host, owner, repo, sha), nil
}

func convertGithubPR(ghPR github.PullRequest) gitprovider.PullRequest {
	pr := gitprovider.PullRequest{
		Number:         int64(ptr.Deref(ghPR.Number, 0)),
		URL:            ptr.Deref(ghPR.HTMLURL, ""),
		Open:           ptr.Deref(ghPR.State, prStateClosed) == prStateOpen,
		Merged:         ghPR.MergedAt != nil,
		MergeCommitSHA: ptr.Deref(ghPR.MergeCommitSHA, ""),
		Object:         ghPR,
		HeadSHA:        ptr.Deref(ghPR.Head.SHA, ""),
	}
	if ghPR.CreatedAt != nil {
		pr.CreatedAt = &ghPR.CreatedAt.Time
	}
	return pr
}
