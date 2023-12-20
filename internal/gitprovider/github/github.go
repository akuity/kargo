package github

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/go-github/v56/github"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"

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
		NewService: func() (gitprovider.GitProviderService, error) {
			return NewGitHubProvider()
		},
	}
)

func init() {
	gitprovider.RegisterProvider(GitProviderServiceName, githubRegistration)
}

type GitHubProvider struct { // nolint: revive
	client *github.Client
}

func NewGitHubProvider() (gitprovider.GitProviderService, error) {
	client := github.NewClient(nil)
	return &GitHubProvider{
		client: client,
	}, nil
}

func (g *GitHubProvider) WithAuthToken(token string) gitprovider.GitProviderService {
	g.client = g.client.WithAuthToken(token)
	return g
}

func (g *GitHubProvider) CreatePullRequest(
	ctx context.Context,
	repoURL string,
	opts gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, err
	}

	ghPR, _, err := g.client.PullRequests.Create(ctx,
		owner,
		repo,
		&github.NewPullRequest{
			Title:               &opts.Title,
			Head:                &opts.Head,
			Base:                &opts.Base,
			Body:                &opts.Title,
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
	repoURL string,
	id int64,
) (*gitprovider.PullRequest, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, err
	}
	ghPR, _, err := g.client.PullRequests.Get(ctx, owner, repo, int(id))
	if err != nil {
		return nil, err
	}
	return convertGithubPR(ghPR), nil
}

func (g *GitHubProvider) ListPullRequests(
	ctx context.Context,
	repoURL string,
	opts gitprovider.ListPullRequestOpts,
) ([]*gitprovider.PullRequest, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, err
	}
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
	ghPRs, _, err := g.client.PullRequests.List(ctx, owner, repo, &listOpts)
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
	switch pointer.StringDeref(ghPR.State, "") {
	case "open":
		prState = gitprovider.PullRequestStateOpen
	case "closed":
		prState = gitprovider.PullRequestStateClosed
	}
	return &gitprovider.PullRequest{
		Number:         int64(pointer.IntDeref(ghPR.Number, 0)),
		URL:            pointer.StringDeref(ghPR.HTMLURL, ""),
		State:          prState,
		MergeCommitSHA: pointer.StringDeref(ghPR.MergeCommitSHA, ""),
		Object:         ghPR,
	}
}

func parseGitHubURL(u string) (string, string, error) {
	regex := regexp.MustCompile(`^https\://github\.com/([\w-]+)/([\w-]+).*`)
	parts := regex.FindStringSubmatch(u)
	if len(parts) != 3 {
		return "", "", errors.Errorf("error parsing github repository URL %q", u)
	}
	return parts[1], parts[2], nil
}

func (g *GitHubProvider) IsPullRequestMerged(ctx context.Context, repoURL string, id int64) (bool, error) {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return false, err
	}
	// https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#check-if-a-pull-request-has-been-merged
	merged, _, err := g.client.PullRequests.IsMerged(ctx, owner, repo, int(id))
	if err != nil {
		return false, err
	}
	return merged, nil
}
