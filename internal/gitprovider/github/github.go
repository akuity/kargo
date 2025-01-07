package github

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v56/github"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/gitprovider"
)

const ProviderName = "github"

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
	host, owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	client := github.NewClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureSkipTLSVerify, // nolint: gosec
			},
		},
	})
	if host != "github.com" {
		baseURL := fmt.Sprintf("https://%s", host)
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
			MaintainerCanModify: github.Bool(false),
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
		listOpts.State = "all"
	case gitprovider.PullRequestStateClosed:
		listOpts.State = "closed"
	case gitprovider.PullRequestStateOpen:
		listOpts.State = "open"
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}
	prs := []gitprovider.PullRequest{}
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

func convertGithubPR(ghPR github.PullRequest) gitprovider.PullRequest {
	pr := gitprovider.PullRequest{
		Number:         int64(ptr.Deref(ghPR.Number, 0)),
		URL:            ptr.Deref(ghPR.HTMLURL, ""),
		Open:           ptr.Deref(ghPR.State, "closed") == "open",
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

func parseRepoURL(repoURL string) (string, string, string, error) {
	u, err := url.Parse(git.NormalizeURL(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing github repository URL %q: %w", u, err)
	}
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("could not extract repository owner and name from URL %q", u)
	}
	return u.Host, parts[0], parts[1], nil
}
