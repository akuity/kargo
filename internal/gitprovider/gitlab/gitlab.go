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

const ProviderName = "gitlab"

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// We assume that any hostname with the word "gitlab" in it, can use this
		// provider. NOTE: We will miss cases where the host is self-hosted Gitlab
		// but doesn't incorporate the word "gitlab" in the hostname. e.g.
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

// provider is a GitLab-based implementation of gitprovider.Interface.
type provider struct { // nolint: revive
	projectName string
	client      mergeRequestClient
}

// NewProvider returns a GitLab-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}
	host, projectName, err := parseRepoURL(repoURL)
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
	return &provider{
		projectName: projectName,
		client:      client.MergeRequests,
	}, nil
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	_ context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.CreatePullRequestOpts{}
	}
	glMR, _, err := p.client.CreateMergeRequest(p.projectName, &gitlab.CreateMergeRequestOptions{
		Title:              &opts.Title,
		Description:        &opts.Description,
		SourceBranch:       &opts.Head,
		TargetBranch:       &opts.Base,
		RemoveSourceBranch: gitlab.Ptr(true),
	})
	if err != nil {
		return nil, err
	}
	if glMR == nil {
		return nil, fmt.Errorf("unexpected nil merge request")
	}
	pr := convertGitlabMR(*glMR)
	return &pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	glMR, _, err := p.client.GetMergeRequest(p.projectName, int(id), nil)
	if err != nil {
		return nil, err
	}
	if glMR == nil {
		return nil, fmt.Errorf("unexpected nil merge request")
	}
	pr := convertGitlabMR(*glMR)
	return &pr, nil
}

// ListPullRequests implements gitprovider.Interface.
func (p *provider) ListPullRequests(
	_ context.Context,
	opts *gitprovider.ListPullRequestOptions,
) ([]gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.ListPullRequestOptions{}
	}
	if opts.State == "" {
		opts.State = gitprovider.PullRequestStateOpen
	}
	switch opts.State {
	case gitprovider.PullRequestStateAny, gitprovider.PullRequestStateClosed,
		gitprovider.PullRequestStateOpen:
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}
	listOpts := &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: &opts.HeadBranch,
		TargetBranch: &opts.BaseBranch,
		ListOptions: gitlab.ListOptions{
			// The max isn't documented, but this doesn't produce an error.
			PerPage: 100,
		},
	}
	prs := []gitprovider.PullRequest{}
	for {
		glMRs, res, err := p.client.ListProjectMergeRequests(p.projectName, listOpts)
		if err != nil {
			return nil, err
		}
		for _, glMR := range glMRs {
			if (opts.State == gitprovider.PullRequestStateAny ||
				((opts.State == gitprovider.PullRequestStateOpen) == isMROpen(*glMR))) &&
				(opts.HeadCommit == "" || glMR.SHA == opts.HeadCommit) {
				prs = append(prs, convertGitlabMR(*glMR))
			}
		}
		if res == nil || res.NextPage == 0 {
			break
		}
		listOpts.Page = res.NextPage
	}
	return prs, nil
}

func convertGitlabMR(glMR gitlab.MergeRequest) gitprovider.PullRequest {
	fmt.Println(glMR.MergeCommitSHA)
	return gitprovider.PullRequest{
		Number:         int64(glMR.IID),
		URL:            glMR.WebURL,
		Open:           isMROpen(glMR),
		Merged:         glMR.State == "merged",
		MergeCommitSHA: glMR.MergeCommitSHA,
		Object:         glMR,
		HeadSHA:        glMR.SHA,
		CreatedAt:      glMR.CreatedAt,
	}
}

func isMROpen(glMR gitlab.MergeRequest) bool {
	return glMR.State == "opened" || glMR.State == "locked"
}

func parseRepoURL(repoURL string) (string, string, error) {
	u, err := url.Parse(git.NormalizeURL(repoURL))
	if err != nil {
		return "", "", fmt.Errorf("error parsing gitlab repository URL %q: %w", u, err)
	}
	return u.Host, strings.TrimPrefix(u.Path, "/"), nil
}
