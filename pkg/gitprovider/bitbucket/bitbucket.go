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

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
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

// pullRequestClient defines the interface for pull request operations.
type pullRequestClient interface {
	CreatePullRequest(opt *bitbucket.PullRequestsOptions) (any, error)
	ListPullRequests(opt *bitbucket.PullRequestsOptions) (any, error)
	GetPullRequest(opt *bitbucket.PullRequestsOptions) (any, error)
	GetCommit(opt *bitbucket.CommitsOptions) (any, error)
	MergePullRequest(opt *bitbucket.PullRequestsOptions) (any, error)
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
) (p gitprovider.Interface, err error) {
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

	// The go-bitbucket library may call log.Fatalf for certain error conditions,
	// which causes a panic. We recover from this to return a proper error.
	defer func() {
		if r := recover(); r != nil {
			p = nil
			err = fmt.Errorf("failed to create Bitbucket client: %v", r)
		}
	}()

	client := bitbucket.NewOAuthbearerToken(opts.Token)
	client.HttpClient = cleanhttp.DefaultClient()

	return &provider{
		owner:    owner,
		repoSlug: repoSlug,
		client:   &clientWrapper{client},
	}, nil
}

// clientWrapper wraps a bitbucket.Client to implement the prClient interface
type clientWrapper struct {
	client *bitbucket.Client
}

func (w *clientWrapper) CreatePullRequest(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Create(opt)
}

func (w *clientWrapper) ListPullRequests(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Gets(opt)
}

func (w *clientWrapper) GetPullRequest(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Get(opt)
}

func (w *clientWrapper) GetCommit(
	opt *bitbucket.CommitsOptions,
) (any, error) {
	return w.client.Repositories.Commits.GetCommit(opt)
}

func (w *clientWrapper) MergePullRequest(
	opt *bitbucket.PullRequestsOptions,
) (any, error) {
	return w.client.Repositories.PullRequests.Merge(opt)
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

	pr, err := toBitbucketPR(resp)
	if err != nil {
		return nil, err
	}

	if pr.MergeCommit.Hash != "" {
		fullSHA, err := p.getFullCommitSHA(ctx, pr.MergeCommit.Hash)
		if err != nil {
			return nil, err
		}
		pr.MergeCommit.Hash = fullSHA
	}

	return toProviderPR(pr, resp), nil
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

	pr, err := toBitbucketPR(resp)
	if err != nil {
		return nil, err
	}

	if pr.MergeCommit.Hash != "" {
		fullSHA, err := p.getFullCommitSHA(ctx, pr.MergeCommit.Hash)
		if err != nil {
			return nil, err
		}
		pr.MergeCommit.Hash = fullSHA
	}

	return toProviderPR(pr, resp), nil
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
		return nil, fmt.Errorf("list response missing 'values' field")
	}

	prList, ok := rawPRs.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for list values: %T", rawPRs)
	}

	// NB: The Bitbucket API doesn't support filtering by source/destination
	// branch or commit hash, so we have to filter client-side, which is
	// highly inefficient.
	var prs []gitprovider.PullRequest
	for _, pr := range prList {
		bbPR, err := toBitbucketPR(pr)
		if err != nil {
			continue
		}

		if opts.HeadBranch != "" && bbPR.Source.Branch.Name != opts.HeadBranch {
			continue
		}

		if opts.BaseBranch != "" && bbPR.Destination.Branch.Name != opts.BaseBranch {
			continue
		}

		if opts.HeadCommit != "" && bbPR.Source.Commit.Hash != opts.HeadCommit {
			continue
		}

		if bbPR.MergeCommit.Hash != "" {
			fullSHA, err := p.getFullCommitSHA(ctx, bbPR.MergeCommit.Hash)
			if err != nil {
				return nil, err
			}
			bbPR.MergeCommit.Hash = fullSHA
		}

		if converted := toProviderPR(bbPR, pr); converted != nil {
			prs = append(prs, *converted)
		}
	}
	return prs, nil
}

// MergePullRequest implements gitprovider.Interface.
func (p *provider) MergePullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, bool, error) {
	// Get the PR to check its state
	prOpts := &bitbucket.PullRequestsOptions{
		Owner:    p.owner,
		RepoSlug: p.repoSlug,
		ID:       strconv.FormatInt(id, 10),
	}

	prResp, err := p.client.GetPullRequest(prOpts)
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}

	bbPR, err := toBitbucketPR(prResp)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing pull request response: %w", err)
	}

	// Check if PR is already merged
	if bbPR.State == prStateMerged {
		return toProviderPR(bbPR, prResp), true, nil
	}

	// Check if PR is closed (any non-open state except merged)
	if bbPR.State != prStateOpen {
		// If it's not open and not merged, then it's closed
		return nil, false, fmt.Errorf(
			"pull request %d is closed but not merged (state: %s)", id, bbPR.State,
		)
	}

	// Check if PR is draft - cannot merge draft PRs
	if bbPR.Draft {
		return nil, false, nil
	}

	// TODO: The Bitbucket API lacks comprehensive merge eligibility checks. We
	// cannot reliably determine if a PR is mergeable due to conflicts, failing
	// checks, or other blocking conditions before attempting the merge. This
	// means we have no choice but to attempt the merge and hope for the best.
	//
	// See: https://jira.atlassian.com/browse/BCLOUD-22014
	//
	// This limitation makes the "wait" option unreliable for Bitbucket
	// repositories.

	// Attempt to merge the PR
	mergeOpts := &bitbucket.PullRequestsOptions{
		Owner:    p.owner,
		RepoSlug: p.repoSlug,
		ID:       strconv.FormatInt(id, 10),
	}

	// Perform the merge
	mergeResp, err := p.client.MergePullRequest(mergeOpts)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}

	// Parse the merged PR response
	mergedBBPR, err := toBitbucketPR(mergeResp)
	if err != nil {
		return nil, false, fmt.Errorf("error parsing merged pull request response: %w", err)
	}

	return toProviderPR(mergedBBPR, mergeResp), true, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(repoURL string, sha string) (string, error) {
	normalizedURL := urls.NormalizeGit(repoURL)

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}

	commitURL := fmt.Sprintf("https://%s%s/commits/%s", parsedURL.Host, parsedURL.Path, sha)

	return commitURL, nil
}

func (p *provider) getFullCommitSHA(ctx context.Context, shortSHA string) (string, error) {
	if shortSHA == "" {
		return "", nil
	}

	commitOpts := &bitbucket.CommitsOptions{
		Owner:    p.owner,
		RepoSlug: p.repoSlug,
		Revision: shortSHA,
	}
	commitOpts.WithContext(ctx)

	resp, err := p.client.GetCommit(commitOpts)
	if err != nil {
		return "", err
	}

	commitResp, ok := resp.(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected commit response type: %T", resp)
	}

	hash, ok := commitResp["hash"].(string)
	if !ok || hash == "" {
		return "", fmt.Errorf("commit response missing 'hash' field")
	}
	return hash, nil
}

// bitbucketPR represents the structure of a Bitbucket pull request.
// See: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/#api-repositories-workspace-repo-slug-pullrequests-pull-request-id-get
// nolint:lll
type bitbucketPR struct {
	ID    int64  `json:"id"`
	State string `json:"state"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"destination"`
	MergeCommit struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	Draft     bool   `json:"draft"`
	CreatedOn string `json:"created_on"`
}

// toBitbucketPR converts a raw response to a bitbucketPR type.
func toBitbucketPR(resp any) (*bitbucketPR, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal PR response: %w", err)
	}
	var pr bitbucketPR
	if err = json.Unmarshal(b, &pr); err != nil {
		return nil, fmt.Errorf("unmarshal PR response: %w", err)
	}
	return &pr, nil
}

// toProviderPR converts a bitbucketPR to a gitprovider.PullRequest.
func toProviderPR(pr *bitbucketPR, raw any) *gitprovider.PullRequest {
	if pr == nil {
		return nil
	}

	var createdAt *time.Time
	if ts, err := time.Parse("2006-01-02T15:04:05Z", pr.CreatedOn); err == nil {
		createdAt = &ts
	}

	return &gitprovider.PullRequest{
		Number: pr.ID,
		URL:    pr.Links.HTML.Href,
		Open:   pr.State == prStateOpen,
		Merged: pr.State == prStateMerged,
		// NB: As a sign of true craftsmanship, or lack thereof, the Bitbucket
		// API returns a short commit SHA as merge commit hash. To get the full
		// commit SHA, we need to fetch the commit details separately.
		MergeCommitSHA: pr.MergeCommit.Hash,
		HeadSHA:        pr.Source.Commit.Hash,
		CreatedAt:      createdAt,
		Object:         raw,
	}
}

// parseRepoURL extracts host, owner and repo slug from a repository URL
func parseRepoURL(repoURL string) (host, owner, slug string, err error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("parse Bitbucket URL %q: %w", repoURL, err)
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid repository path in URL %q", u)
	}

	return u.Hostname(), parts[0], parts[1], nil
}
