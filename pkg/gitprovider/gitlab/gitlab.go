package gitlab

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	ProviderName = "gitlab"

	mrStateOpened     = "opened"
	mrStateMerged     = "merged"
	mrStateSkipMerged = "skip_merged"
	mrStateLocked     = "locked"
)

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
	) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error)

	GetMergeRequest(
		pid any,
		mergeRequest int64,
		opt *gitlab.GetMergeRequestsOptions,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeRequest, *gitlab.Response, error)

	AcceptMergeRequest(
		pid any,
		mergeRequest int64,
		opt *gitlab.AcceptMergeRequestOptions,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeRequest, *gitlab.Response, error)
}

type mergeTrainClient interface {
	GetMergeRequestOnAMergeTrain(
		pid any,
		mergeRequest int64,
		options ...gitlab.RequestOptionFunc,
	) (*gitlab.MergeTrain, *gitlab.Response, error)
}

// provider is a GitLab-based implementation of gitprovider.Interface.
type provider struct { // nolint: revive
	projectName      string
	client           mergeRequestClient
	mergeTrainClient mergeTrainClient
}

// NewProvider returns a GitLab-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}

	scheme, host, projectName, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	clientOpts := make([]gitlab.ClientOptionFunc, 0, 2)

	if host != "gitlab.com" {
		clientOpts = append(
			clientOpts,
			gitlab.WithBaseURL(fmt.Sprintf("%s://%s/api/v4", scheme, host)),
		)
	}

	httpClient := cleanhttp.DefaultClient()
	if opts.InsecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}
	clientOpts = append(clientOpts, gitlab.WithHTTPClient(httpClient))

	client, err := gitlab.NewClient(opts.Token, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &provider{
		projectName:      projectName,
		client:           client.MergeRequests,
		mergeTrainClient: client.MergeTrains,
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
		Title:        &opts.Title,
		Description:  &opts.Description,
		Labels:       (*gitlab.LabelOptions)(&opts.Labels),
		SourceBranch: &opts.Head,
		TargetBranch: &opts.Base,
	})
	if err != nil {
		return nil, err
	}
	if glMR == nil {
		return nil, fmt.Errorf("unexpected nil merge request")
	}
	pr := convertGitlabMR(glMR.BasicMergeRequest)
	return &pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	glMR, _, err := p.client.GetMergeRequest(p.projectName, id, nil)
	if err != nil {
		return nil, err
	}
	if glMR == nil {
		return nil, fmt.Errorf("unexpected nil merge request")
	}
	pr := convertGitlabMR(glMR.BasicMergeRequest)
	// Check if MR is in merge train when state is opened
	if pr.Open {
		mergeTrain, mtResp, err := p.mergeTrainClient.GetMergeRequestOnAMergeTrain(
			p.projectName, id,
		)
		if err == nil && mtResp != nil && mtResp.StatusCode == 200 && mergeTrain != nil {
			if mergeTrain.Status != mrStateMerged && mergeTrain.Status != mrStateSkipMerged {
				pr.Queued = true
			}
		} else if mtResp != nil && mtResp.StatusCode == 403 {
			// Merge Trains not available (Free tier), fall back to state check
			pr.Queued = true
		}
		// HTTP 404 means not in merge train, leave Queued as false
	}
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
	var prs []gitprovider.PullRequest
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

// MergePullRequest implements gitprovider.Interface.
func (p *provider) MergePullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, bool, error) {
	glMR, _, err := p.client.GetMergeRequest(p.projectName, id, nil)
	if err != nil {
		return nil, false, fmt.Errorf("error getting merge request %d: %w", id, err)
	}
	if glMR == nil {
		return nil, false, fmt.Errorf("merge request %d not found", id)
	}

	switch {
	case glMR.State == mrStateMerged:
		pr := convertGitlabMR(glMR.BasicMergeRequest)
		return &pr, true, nil

	case glMR.State != mrStateOpened:
		return nil, false, fmt.Errorf("pull request %d is closed but not merged", id)

	case glMR.DetailedMergeStatus != "mergeable":
		return nil, false, nil
	}

	// Merge the MR
	updatedMR, resp, err := p.client.AcceptMergeRequest(
		p.projectName, id, &gitlab.AcceptMergeRequestOptions{},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging merge request %d: %w", id, err)
	}
	if updatedMR == nil {
		return nil, false, fmt.Errorf("unexpected nil merge request after merge")
	}

	pr := convertGitlabMR(updatedMR.BasicMergeRequest)
	if pr.Merged {
		return &pr, true, nil
	}

	// GitLab merge trains keep the MR in "opened" state when queued for merging.
	// The GitLab API returns HTTP 200 with the MR object in "opened" state.
	if resp != nil && updatedMR.State == mrStateOpened {
		// Verify the MR is actually in the merge train using the Merge Trains API.
		// This API is only available in GitLab Premium/Ultimate tiers.
		mergeTrain, mtResp, err := p.mergeTrainClient.GetMergeRequestOnAMergeTrain(
			p.projectName, id,
		)
		if err == nil && mtResp != nil && mtResp.StatusCode == 200 && mergeTrain != nil {
			// MR is confirmed to be in the merge train. Check if it's in an active state.
			// Status can be: idle, merged, stale, fresh, merging, skip_merged
			// We consider it queued if it's in an active state (not merged/skip_merged).
			if mergeTrain.Status != mrStateMerged && mergeTrain.Status != mrStateSkipMerged {
				pr.Queued = true
			}
		} else if mtResp != nil && mtResp.StatusCode == 403 {
			// Merge Trains API is not available (Free tier or feature disabled).
			// Fall back to the simple state check: if AcceptMergeRequest succeeded
			// and state is still "opened", assume it's queued.
			pr.Queued = true
		}
		// If we get 404, the MR is not in the merge train, so Queued remains false.
	}

	// MR is not merged yet (queued or pending checks). Return non-merged so
	// the caller can decide to wait/retry according to its policy.
	return &pr, false, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(repoURL string, sha string) (string, error) {
	normalizedURL := urls.NormalizeGit(repoURL)

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}

	commitURL := fmt.Sprintf("https://%s%s/-/commit/%s", parsedURL.Host, parsedURL.Path, sha)

	return commitURL, nil
}

func convertGitlabMR(glMR gitlab.BasicMergeRequest) gitprovider.PullRequest {
	return gitprovider.PullRequest{
		Number:         glMR.IID,
		URL:            glMR.WebURL,
		Open:           isMROpen(glMR),
		Merged:         glMR.State == mrStateMerged,
		MergeCommitSHA: glMR.MergeCommitSHA,
		Object:         glMR,
		HeadSHA:        glMR.SHA,
		CreatedAt:      glMR.CreatedAt,
	}
}

func isMROpen(glMR gitlab.BasicMergeRequest) bool {
	return glMR.State == mrStateOpened || glMR.State == mrStateLocked
}

func parseRepoURL(repoURL string) (string, string, string, error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf(
			"error parsing gitlab repository URL %q: %w", u, err,
		)
	}

	scheme := u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}

	return scheme, u.Host, strings.TrimPrefix(u.Path, "/"), nil
}
