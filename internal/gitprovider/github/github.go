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

const (
	GitProviderServiceName = "github"
)

var (
	githubRegistration = gitprovider.ProviderRegistration{
		Predicate: func(repoURL string) bool {
			u, err := url.Parse(repoURL)
			if err != nil {
				return false
			}
			// We assume that any hostname with the word "github" in the hostname,
			// can use this provider. NOTE: we will miss cases where the host is
			// GitHub but doesn't incorporate the word "github" in the hostname.
			// e.g. 'git.mycompany.com'
			return strings.Contains(u.Host, GitProviderServiceName)
		},
		NewService: func(
			repoURL string,
			opts *gitprovider.GitProviderOptions,
		) (gitprovider.GitProviderService, error) {
			return NewGitHubProvider(repoURL, opts)
		},
	}
)

func init() {
	gitprovider.RegisterProvider(GitProviderServiceName, githubRegistration)
}

type GitHubProvider struct { // nolint: revive
	owner  string
	repo   string
	client *github.Client
}

func NewGitHubProvider(
	repoURL string,
	opts *gitprovider.GitProviderOptions,
) (gitprovider.GitProviderService, error) {
	if opts == nil {
		opts = &gitprovider.GitProviderOptions{}
	}
	host, owner, repo, err := parseGitHubURL(repoURL)
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
	return &GitHubProvider{
		owner:  owner,
		repo:   repo,
		client: client,
	}, nil
}

func (g *GitHubProvider) CreatePullRequest(
	ctx context.Context,
	opts gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	ghPR, _, err := g.client.PullRequests.Create(ctx,
		g.owner,
		g.repo,
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
	return convertGithubPR(ghPR), nil
}

func (g *GitHubProvider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	ghPR, _, err := g.client.PullRequests.Get(ctx, g.owner, g.repo, int(id))
	if err != nil {
		return nil, err
	}
	return convertGithubPR(ghPR), nil
}

func (g *GitHubProvider) ListPullRequests(
	ctx context.Context,
	opts gitprovider.ListPullRequestOpts,
) ([]*gitprovider.PullRequest, error) {
	listOpts := github.PullRequestListOptions{
		Head: opts.Head,
		Base: opts.Base,
	}
	switch opts.State {
	case "", gitprovider.PullRequestStateOpen:
		listOpts.State = "open"
	case gitprovider.PullRequestStateClosed:
		listOpts.State = "closed"
	}
	ghPRs, _, err := g.client.PullRequests.List(ctx, g.owner, g.repo, &listOpts)
	if err != nil {
		return nil, err
	}
	prs := make([]*gitprovider.PullRequest, len(ghPRs))
	for i, ghPR := range ghPRs {
		prs[i] = convertGithubPR(ghPR)
	}
	return prs, nil
}

func convertGithubPR(ghPR *github.PullRequest) *gitprovider.PullRequest {
	var prState gitprovider.PullRequestState
	switch ptr.Deref(ghPR.State, "") {
	case "open":
		prState = gitprovider.PullRequestStateOpen
	case "closed":
		prState = gitprovider.PullRequestStateClosed
	}
	pr := &gitprovider.PullRequest{
		Number:         int64(ptr.Deref(ghPR.Number, 0)),
		URL:            ptr.Deref(ghPR.HTMLURL, ""),
		State:          prState,
		MergeCommitSHA: ptr.Deref(ghPR.MergeCommitSHA, ""),
		Object:         ghPR,
		HeadSHA:        ptr.Deref(ghPR.Head.SHA, ""),
	}
	if ghPR.CreatedAt != nil {
		pr.CreatedAt = &ghPR.CreatedAt.Time
	}
	return pr
}

func parseGitHubURL(repoURL string) (string, string, string, error) {
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

func (g *GitHubProvider) IsPullRequestMerged(ctx context.Context, id int64) (bool, error) {
	// https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#check-if-a-pull-request-has-been-merged
	merged, _, err := g.client.PullRequests.IsMerged(ctx, g.owner, g.repo, int(id))
	if err != nil {
		return false, err
	}
	return merged, nil
}
