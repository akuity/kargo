package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/ktrysmt/go-bitbucket"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/gitprovider"
)

const (
	ProviderName = "bitbucket"

	// supportedHost is the hostname of the Bitbucket instance that this provider
	// supports. As of now, this provider only supports Bitbucket "Cloud", and not
	// self-hosted Bitbucket "Datacenter" instances.
	supportedHost = "bitbucket.org"

	// prStateOpen is the state of an open pull request.
	prStateOpen = "OPEN"
	// prStateMerged is the state of a merged pull request.
	prStateMerged = "MERGED"
	// prStateDeclined is the state of a declined pull request. This is also
	// known as "closed" in other Git providers.
	prStateDeclined = "DECLINED"
	// prStateSuperseded is the state of a superseded pull request. This is also
	// known as "closed" in other Git providers.
	prStateSuperseded = "SUPERSEDED"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		return u.Hostname() == supportedHost
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
	) (any, error)

	ListPullRequests(
		opt *bitbucket.PullRequestsOptions,
	) (any, error)

	GetPullRequest(
		opt *bitbucket.PullRequestsOptions,
	) (any, error)
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

	// The provider only supports Bitbucket "Cloud", and not self-hosted
	// Bitbucket "Datacenter" instances — these require a different API client.
	if host != supportedHost {
		return nil, fmt.Errorf("unsupported Bitbucket host %q", host)
	}

	client := bitbucket.NewOAuthbearerToken(opts.Token)
	client.HttpClient = cleanhttp.DefaultClient()

	return &provider{
		owner:    owner,
		repoSlug: repoSlug,
		client:   &bitbucketClientWrapper{client},
	}, nil
}

type bitbucketClientWrapper struct {
	client *bitbucket.Client
}

func (w *bitbucketClientWrapper) CreatePullRequest(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Create(opt)
}

func (w *bitbucketClientWrapper) ListPullRequests(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Gets(opt)
}

func (w *bitbucketClientWrapper) GetPullRequest(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Get(opt)
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	ctx context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.CreatePullRequestOpts{}
	}

	createOpts := &bitbucket.PullRequestsOptions{
		Owner:             p.owner,
		RepoSlug:          p.repoSlug,
		Title:             opts.Title,
		Description:       opts.Description,
		SourceBranch:      opts.Head,
		DestinationBranch: opts.Base,
	}
	createOpts.WithContext(ctx)

	resp, err := p.client.CreatePullRequest(createOpts)
	if err != nil {
		return nil, err
	}

	pr := convertBitbucketPR(resp)
	return &pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	getOpts := &bitbucket.PullRequestsOptions{
		Owner:    p.owner,
		RepoSlug: p.repoSlug,
		ID:       strconv.FormatInt(id, 10),
	}
	getOpts.WithContext(ctx)

	resp, err := p.client.GetPullRequest(getOpts)
	if err != nil {
		return nil, err
	}

	pr := convertBitbucketPR(resp)
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

	listOpts := &bitbucket.PullRequestsOptions{
		Owner:    p.owner,
		RepoSlug: p.repoSlug,
		// TODO(hidde): Listing pull requests does not support filtering by
		// source or destination branch. This is a limitation of the Bitbucket
		// API. Because of this, filtering will have to be done client-side (in
		// other words, further down in this method), and will be highly
		// inefficient.
	}
	listOpts.WithContext(ctx)

	switch opts.State {
	case gitprovider.PullRequestStateAny:
		listOpts.States = []string{prStateOpen, prStateMerged, prStateDeclined, prStateSuperseded}
	case gitprovider.PullRequestStateClosed:
		listOpts.States = []string{prStateMerged, prStateDeclined, prStateSuperseded}
	case gitprovider.PullRequestStateOpen:
		listOpts.States = []string{prStateOpen}
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}

	resp, err := p.client.ListPullRequests(listOpts)
	if err != nil {
		return nil, err
	}

	list, ok := resp.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for list response: %T", resp)
	}

	rawPRs, ok := list["values"]
	if !ok {
		return nil, fmt.Errorf("list response does not contain %q", "values")
	}

	prList, ok := rawPRs.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for list values: %T", rawPRs)
	}

	var prs []gitprovider.PullRequest
	for _, pr := range prList {
		prs = append(prs, convertBitbucketPR(pr))
	}
	return prs, nil
}

// pullRequest is the (partial) structure of a Bitbucket pull request.
// xref: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/#api-repositories-workspace-repo-slug-pullrequests-pull-request-id-get
type pullRequest struct {
	ID    int64  `json:"id"`
	State string `json:"state"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Source struct {
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"source"`
	MergeCommit struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	CreatedOn string `json:"created_on"`
}

func convertBitbucketPR(raw any) gitprovider.PullRequest {
	b, err := json.Marshal(raw)
	if err != nil {
		return gitprovider.PullRequest{}
	}

	var typedPR pullRequest
	if err := json.Unmarshal(b, &typedPR); err != nil {
		return gitprovider.PullRequest{}
	}

	var createdAt *time.Time
	if ts, err := time.Parse("2006-01-02T15:04:05Z", typedPR.CreatedOn); err == nil {
		createdAt = &ts
	}

	return gitprovider.PullRequest{
		Number: typedPR.ID,
		URL:    typedPR.Links.HTML.Href,
		Open:   typedPR.State == prStateOpen,
		Merged: typedPR.State == prStateMerged,
		// TODO(hidde): As a sign of true craftsmanship, or lack thereof, the
		// Bitbucket API returns a short commit SHA as merge commit hash. To get
		// the full commit SHA, we need to fetch the commit details separately.
		MergeCommitSHA: typedPR.MergeCommit.Hash,
		Object:         raw,
		HeadSHA:        typedPR.Source.Commit.Hash,
		CreatedAt:      createdAt,
	}
}

func parseRepoURL(repoURL string) (string, string, string, error) {
	u, err := url.Parse(git.NormalizeURL(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing bitbucket repository URL %q: %w", u, err)
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf(
			"could not extract repository owner and slug from URL %q", u,
		)
	}

	return u.Hostname(), parts[0], parts[1], nil
}
