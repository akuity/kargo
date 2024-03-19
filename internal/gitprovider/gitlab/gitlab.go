package gitlab

import (
	"context"
	"net/url"
	"strings"

	"github.com/xanzy/go-gitlab"

	"github.com/akuity/kargo/internal/gitprovider"
)

const (
	GitProviderServiceName = "gitlab"
)

var (
	registration = gitprovider.ProviderRegistration{
		Predicate: func(repoURL string) bool {
			u, err := url.Parse(repoURL)
			if err != nil {
				return false
			}
			// We assume that any hostname with the word "gitlab" in the hostname,
			// can use this provider. NOTE: we will miss cases where the host is
			// Gitlab but doesn't incorporate the word "gitlab" in the hostname.
			// e.g. 'git.mycompany.com'
			return strings.Contains(u.Host, GitProviderServiceName)
		},
		NewService: func() (gitprovider.GitProviderService, error) {
			return NewGitLabProvider()
		},
	}
)

func init() {
	gitprovider.RegisterProvider(GitProviderServiceName, registration)
}

type MergeRequestClient interface {
	CreateMergeRequest(pid interface{}, opt *gitlab.CreateMergeRequestOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
	ListProjectMergeRequests(pid interface{}, opt *gitlab.ListProjectMergeRequestsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.MergeRequest, *gitlab.Response, error)
	GetMergeRequest(pid interface{}, mergeRequest int, opt *gitlab.GetMergeRequestsOptions, options ...gitlab.RequestOptionFunc) (*gitlab.MergeRequest, *gitlab.Response, error)
}

type GitLabClient struct {
	MergeRequests MergeRequestClient
}

type GitLabProvider struct { // nolint: revive
	client *GitLabClient
}

func NewGitLabProvider() (gitprovider.GitProviderService, error) {
	client, err := gitlab.NewClient("")
	if err != nil {
		return nil, err
	}
	return &GitLabProvider{
		client: &GitLabClient{MergeRequests: client.MergeRequests},
	}, nil
}

func (g *GitLabProvider) WithAuthToken(token string) (gitprovider.GitProviderService, error) {
	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}
	g.client = &GitLabClient{MergeRequests: client.MergeRequests}
	return g, nil
}

func (g *GitLabProvider) CreatePullRequest(
	ctx context.Context,
	repoURL string,
	opts gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	projectName, err := getProjectNameFromUrl(repoURL)
	if err != nil {
		return nil, err
	}

	glMR, _, err := g.client.MergeRequests.CreateMergeRequest(projectName, &gitlab.CreateMergeRequestOptions{
		Title:              &opts.Title,
		Description:        &opts.Description,
		SourceBranch:       &opts.Head,
		TargetBranch:       &opts.Base,
		RemoveSourceBranch: gitlab.Ptr(true),
	})
	if err != nil {
		return nil, err
	}
	return convertGitlabMR(glMR), nil
}

func (g *GitLabProvider) GetPullRequest(
	ctx context.Context,
	repoURL string,
	id int64,
) (*gitprovider.PullRequest, error) {
	glMR, err := g.getMergeRequest(repoURL, id)
	if err != nil {
		return nil, err
	}
	return convertGitlabMR(glMR), nil
}

func (g *GitLabProvider) ListPullRequests(
	ctx context.Context,
	repoURL string,
	opts gitprovider.ListPullRequestOpts,
) ([]*gitprovider.PullRequest, error) {
	projectName, err := getProjectNameFromUrl(repoURL)
	if err != nil {
		return nil, err
	}
	listOpts := &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &opts.Head,
		TargetBranch: &opts.Base,
	}
	glMRs, _, err := g.client.MergeRequests.ListProjectMergeRequests(projectName, listOpts)
	if err != nil {
		return nil, err
	}
	prs := make([]*gitprovider.PullRequest, 0, len(glMRs))
	for _, glMR := range glMRs {
		if (opts.State == gitprovider.PullRequestStateOpen) == isMROpen(glMR) {
			prs = append(prs, convertGitlabMR(glMR))
		}
	}
	return prs, nil
}

func (g *GitLabProvider) IsPullRequestMerged(ctx context.Context, repoURL string, id int64) (bool, error) {
	glMR, err := g.getMergeRequest(repoURL, id)
	if err != nil {
		return false, err
	}
	return glMR.State == "merged", nil
}

func convertGitlabMR(glMR *gitlab.MergeRequest) *gitprovider.PullRequest {
	var prState gitprovider.PullRequestState
	if isMROpen(glMR) {
		prState = gitprovider.PullRequestStateOpen
	} else {
		prState = gitprovider.PullRequestStateClosed
	}
	return &gitprovider.PullRequest{
		Number:         int64(glMR.IID),
		URL:            glMR.WebURL,
		State:          prState,
		MergeCommitSHA: glMR.MergeCommitSHA,
		Object:         glMR,
	}
}

func isMROpen(glMR *gitlab.MergeRequest) bool {
	return glMR.State == "opened" || glMR.State == "locked"
}

func getProjectNameFromUrl(u string) (string, error) {
	gitlabUrl, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimPrefix(gitlabUrl.Path, "/"), ".git"), nil
}

func (g *GitLabProvider) getMergeRequest(repoURL string, id int64) (*gitlab.MergeRequest, error) {
	projectName, err := getProjectNameFromUrl(repoURL)
	if err != nil {
		return nil, err
	}
	glMR, _, err := g.client.MergeRequests.GetMergeRequest(projectName, int(id), nil)
	return glMR, err
}
