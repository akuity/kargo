package gitlab

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xanzy/go-gitlab"

	"github.com/akuity/kargo/internal/git"
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
		NewService: func(
			repoURL string,
			opts *gitprovider.GitProviderOptions,
		) (gitprovider.GitProviderService, error) {
			return NewGitLabProvider(repoURL, opts)
		},
	}
)

func init() {
	gitprovider.RegisterProvider(GitProviderServiceName, registration)
}

type mergeRequestClient interface {
	CreateMergeRequest(
		pid any,
		opt *gitlab.CreateMergeRequestOptions,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeRequest, *gitlab.Response, error)

	ListProjectMergeRequests(
		pid any,
		opt *gitlab.ListProjectMergeRequestsOptions,
		options ...gitlab.RequestOptionFunc,
	) ([]*gitlab.MergeRequest, *gitlab.Response, error)

	GetMergeRequest(
		pid any,
		mergeRequest int,
		opt *gitlab.GetMergeRequestsOptions,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeRequest, *gitlab.Response, error)
}

type gitLabClient struct { // nolint: revive
	mergeRequests mergeRequestClient
}

type gitLabProvider struct { // nolint: revive
	projectName string
	client      *gitLabClient
}

func NewGitLabProvider(
	repoURL string,
	opts *gitprovider.GitProviderOptions,
) (gitprovider.GitProviderService, error) {
	if opts == nil {
		opts = &gitprovider.GitProviderOptions{}
	}
	host, projectName, err := parseGitLabURL(repoURL)
	if err != nil {
		return nil, err
	}
	clientOpts := make([]gitlab.ClientOptionFunc, 0, 2)
	if host != "gitlab.com" {
		clientOpts = append(
			clientOpts,
			gitlab.WithBaseURL(fmt.Sprintf("https://%s/api/v4", host)),
		)
	}
	if opts.InsecureSkipTLSVerify {
		clientOpts = append(
			clientOpts,
			gitlab.WithHTTPClient(&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true, // nolint: gosec
					},
				},
			}),
		)
	}
	client, err := gitlab.NewClient(opts.Token, clientOpts...)
	if err != nil {
		return nil, err
	}
	return &gitLabProvider{
		projectName: projectName,
		client:      &gitLabClient{mergeRequests: client.MergeRequests},
	}, nil
}

func (g *gitLabProvider) CreatePullRequest(
	_ context.Context,
	opts gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	glMR, _, err := g.client.mergeRequests.CreateMergeRequest(g.projectName, &gitlab.CreateMergeRequestOptions{
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

func (g *gitLabProvider) GetPullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	glMR, err := g.getMergeRequest(id)
	if err != nil {
		return nil, err
	}
	return convertGitlabMR(glMR), nil
}

func (g *gitLabProvider) ListPullRequests(
	_ context.Context,
	opts gitprovider.ListPullRequestOpts,
) ([]*gitprovider.PullRequest, error) {
	listOpts := &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &opts.Head,
		TargetBranch: &opts.Base,
	}
	glMRs, _, err := g.client.mergeRequests.ListProjectMergeRequests(g.projectName, listOpts)
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

func (g *gitLabProvider) IsPullRequestMerged(_ context.Context, id int64) (bool, error) {
	glMR, err := g.getMergeRequest(id)
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
		HeadSHA:        glMR.SHA,
	}
}

func isMROpen(glMR *gitlab.MergeRequest) bool {
	return glMR.State == "opened" || glMR.State == "locked"
}

func (g *gitLabProvider) getMergeRequest(id int64) (*gitlab.MergeRequest, error) {
	glMR, _, err := g.client.mergeRequests.GetMergeRequest(g.projectName, int(id), nil)
	return glMR, err
}

func parseGitLabURL(repoURL string) (string, string, error) {
	u, err := url.Parse(git.NormalizeURL(repoURL))
	if err != nil {
		return "", "", fmt.Errorf("error parsing gitlab repository URL %q: %w", u, err)
	}
	return u.Host, strings.TrimPrefix(u.Path, "/"), nil
}
