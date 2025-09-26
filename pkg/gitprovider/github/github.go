package github

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v71/github"
	"github.com/hashicorp/go-cleanhttp"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
)

const ProviderName = "github"

// GitHub pull request states
const (
	prStateAll    = "all"
	prStateClosed = "closed"
	prStateOpen   = "open"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// We assume that any hostname with the word "github" in it can use this
		// provider. NOTE: We will miss cases where the host is GitHub Enterprise
		// but doesn't incorporate the word "github" in the hostname. e.g.
		// 'git.mycompany.com'
		return strings.Contains(u.Host, ProviderName)
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

	scheme, host, owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	httpClient := cleanhttp.DefaultClient()
	if opts.InsecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	client := github.NewClient(httpClient)

	if host != "github.com" {
		baseURL := fmt.Sprintf("%s://%s", scheme, host)
		// This function call will automatically add correct paths to the base URL
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, err
		}
	}
	if opts.Token != "" {
		client = client.WithAuthToken(opts.Token)
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
) (*gitprovider.PullRequest, bool, error) {
	// Get the current PR to check its status
	ghPR, _, err := p.client.GetPullRequests(ctx, p.owner, p.repo, int(id))
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}

	if ghPR == nil {
		return nil, false, fmt.Errorf("pull request %d not found", id)
	}

	// Check if PR is already merged
	if ghPR.MergedAt != nil {
		pr := convertGithubPR(*ghPR)
		return &pr, true, nil
	}

	// Check if PR is closed but not merged - this is a terminal error
	if ptr.Deref(ghPR.State, prStateClosed) != prStateOpen {
		return nil, false, fmt.Errorf("pull request %d is closed but not merged", id)
	}

	mergeResult, _, err := p.client.MergePullRequest(
		ctx,
		p.owner,
		p.repo,
		int(id),
		"", // Empty string - let GitHub use its default Commit Message
		&github.PullRequestOptions{},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}

	if mergeResult == nil {
		return nil, false, fmt.Errorf("unexpected nil merge result")
	}

	// After merging, get the updated PR to return current state
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
	normalizedURL := urls.NormalizeGit(repoURL)

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}

	commitURL := fmt.Sprintf("https://%s%s/commit/%s", parsedURL.Host, parsedURL.Path, sha)

	return commitURL, nil
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

func parseRepoURL(repoURL string) (string, string, string, string, error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", "", fmt.Errorf(
			"error parsing github repository URL %q: %w", u, err,
		)
	}

	scheme := u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", "", fmt.Errorf(
			"could not extract repository owner and name from URL %q", u,
		)
	}

	return scheme, u.Host, parts[0], parts[1], nil
}
