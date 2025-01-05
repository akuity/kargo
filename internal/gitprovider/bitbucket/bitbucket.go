package bitbucket

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/gitprovider"
	"strconv"
	"time"

	"github.com/ktrysmt/go-bitbucket"

	"net/http"
	"net/url"
	"strings"
)

const ProviderName = "bitbucket"

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}

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

type pullRequestClient interface {
	CreatePullRequest(
		opt *bitbucket.PullRequestsOptions,
	) (interface{}, error)

	ListPullRequests(
		opt *bitbucket.PullRequestsOptions,
	) (interface{}, error)

	GetPullRequest(
		opt *bitbucket.PullRequestsOptions,
	) (interface{}, error)
}

// provider is a Bitbucket-based implementation of gitprovider.Interface.
type provider struct {
	owner    string
	repoSlug string
	client   pullRequestClient
}

// NewProvider returns a Bitbucket-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}
	host, owner, repoSlug, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	client := bitbucket.NewBasicAuth(opts.Name, opts.Token)
	if host != "bitbucket.org" {
		client.HttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: opts.InsecureSkipTLSVerify,
				},
			},
		}
	}
	return &provider{
		owner:    owner,
		repoSlug: repoSlug,
		client:   &bitbucketClientWrapper{client},
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
	prOpts := &bitbucket.PullRequestsOptions{
		Owner:             p.owner,
		RepoSlug:          p.repoSlug,
		Title:             opts.Title,
		Description:       opts.Description,
		SourceBranch:      opts.Head,
		DestinationBranch: opts.Base,
	}
	bbPR, err := p.client.CreatePullRequest(prOpts)
	if err != nil {
		return nil, err
	}
	pr := convertBitbucketPR(bbPR)
	return &pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	bbPR, err := p.client.GetPullRequest(&bitbucket.PullRequestsOptions{ID: strconv.FormatInt(id, 10)})
	if err != nil {
		return nil, err
	}
	if bbPR == nil {
		return nil, fmt.Errorf("pull request %d not found", id)
	}
	pr := convertBitbucketPR(bbPR)
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
	prOpts := &bitbucket.PullRequestsOptions{
		SourceBranch:      opts.HeadBranch,
		DestinationBranch: opts.BaseBranch,
	}
	bbPRs, err := p.client.ListPullRequests(prOpts)
	if err != nil {
		return nil, err
	}

	// Type assert bbPRs to the correct type
	prList, ok := bbPRs.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for bbPRs: %T", bbPRs)
	}

	var prs []gitprovider.PullRequest
	for _, bbPR := range prList {
		prs = append(prs, convertBitbucketPR(bbPR))
	}
	return prs, nil
}

func convertBitbucketPR(pr interface{}) gitprovider.PullRequest {
	bbPR, ok := pr.(bbPullRequest)
	if !ok {
		return gitprovider.PullRequest{}
	}

	createdOn, err := time.Parse("2006-01-02T15:04:05Z", bbPR.CreatedOn)
	if err != nil {
		fmt.Println("Error parsing CreatedOn:", err)
	}

	return gitprovider.PullRequest{
		Number:         bbPR.ID,
		URL:            bbPR.Links.HTML.Href,
		Open:           bbPR.State == "OPEN",
		Merged:         bbPR.State == "MERGED",
		MergeCommitSHA: bbPR.MergeCommit.Hash,
		Object:         pr,
		HeadSHA:        bbPR.Source.Commit.Hash,
		CreatedAt:      &createdOn,
	}
}

func parseRepoURL(repoURL string) (string, string, string, error) {
	u, err := url.Parse(git.NormalizeURL(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing bitbucket repository URL %q: %w", u, err)
	}
	host := u.Host
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid repository URL %q", repoURL)
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")

	return host, owner, repo, nil
}

type bitbucketClientWrapper struct {
	client *bitbucket.Client
}

func (w *bitbucketClientWrapper) CreatePullRequest(
	opt *bitbucket.PullRequestsOptions,
) (interface{}, error) {
	return w.client.Repositories.PullRequests.Create(opt)
}

func (w *bitbucketClientWrapper) ListPullRequests(
	opt *bitbucket.PullRequestsOptions,
) (interface{}, error) {
	return w.client.Repositories.PullRequests.Gets(opt)
}

func (w *bitbucketClientWrapper) GetPullRequest(
	opt *bitbucket.PullRequestsOptions,
) (interface{}, error) {
	return w.client.Repositories.PullRequests.Get(opt)
}
