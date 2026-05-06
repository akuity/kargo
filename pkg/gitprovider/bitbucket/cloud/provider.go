package cloud

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	// ProviderName is the name used to register the Bitbucket Cloud provider.
	ProviderName = "bitbucket"

	// Host is the hostname of the Bitbucket Cloud instance.
	Host = "bitbucket.org"

	// apiBaseURL is the base URL for the Bitbucket Cloud REST API.
	apiBaseURL = "https://api.bitbucket.org/2.0"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		return u.Hostname() == Host
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

// provider is a Bitbucket Cloud implementation of gitprovider.Interface.
type provider struct {
	owner    string
	repoSlug string
	client   ClientWithResponsesInterface
}

// NewProvider returns a Bitbucket Cloud implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}

	_, owner, repoSlug, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	token := opts.Token
	client, err := NewClientWithResponses(
		apiBaseURL,
		WithHTTPClient(cleanhttp.DefaultClient()),
		WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Bitbucket Cloud API client: %w", err)
	}

	return &provider{
		owner:    owner,
		repoSlug: repoSlug,
		client:   client,
	}, nil
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	ctx context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.CreatePullRequestOpts{}
	}

	title := opts.Title
	body := Pullrequest{Type: "pullrequest", Title: &title}
	if opts.Description != "" {
		body.Set("description", opts.Description)
	}

	srcBranch := opts.Head
	dstBranch := opts.Base
	body.Source = &PullrequestEndpoint{
		Branch: &struct {
			DefaultMergeStrategy *string                                     `json:"default_merge_strategy,omitempty"`
			MergeStrategies      *[]PullrequestEndpointBranchMergeStrategies `json:"merge_strategies,omitempty"`
			Name                 *string                                     `json:"name,omitempty"`
		}{Name: &srcBranch},
	}
	body.Destination = &PullrequestEndpoint{
		Branch: &struct {
			DefaultMergeStrategy *string                                     `json:"default_merge_strategy,omitempty"`
			MergeStrategies      *[]PullrequestEndpointBranchMergeStrategies `json:"merge_strategies,omitempty"`
			Name                 *string                                     `json:"name,omitempty"`
		}{Name: &dstBranch},
	}

	resp, err := p.client.PostRepositoriesWorkspaceRepoSlugPullrequestsWithResponse(
		ctx,
		p.owner,
		p.repoSlug,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating pull request: %w", err)
	}
	if resp.JSON201 == nil {
		return nil, fmt.Errorf(
			"unexpected response %d creating pull request", resp.StatusCode(),
		)
	}
	if err = p.resolveFullMergeCommitSHA(ctx, resp.JSON201); err != nil {
		return nil, err
	}
	return toProviderPR(resp.JSON201), nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	resp, err := p.client.GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdWithResponse(
		ctx,
		p.owner,
		p.repoSlug,
		int(id),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf(
			"unexpected response %d getting pull request %d", resp.StatusCode(), id,
		)
	}
	if err = p.resolveFullMergeCommitSHA(ctx, resp.JSON200); err != nil {
		return nil, err
	}
	return toProviderPR(resp.JSON200), nil
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

	var states []string
	switch opts.State {
	case gitprovider.PullRequestStateAny:
		states = []string{
			string(PullrequestStateOPEN),
			string(PullrequestStateMERGED),
			string(PullrequestStateDECLINED),
			string(PullrequestStateSUPERSEDED),
		}
	case gitprovider.PullRequestStateClosed:
		states = []string{
			string(PullrequestStateMERGED),
			string(PullrequestStateDECLINED),
			string(PullrequestStateSUPERSEDED),
		}
	case gitprovider.PullRequestStateOpen:
		states = []string{string(PullrequestStateOPEN)}
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}

	resp, err := p.client.GetRepositoriesWorkspaceRepoSlugPullrequestsWithResponse(
		ctx,
		p.owner,
		p.repoSlug,
		&GetRepositoriesWorkspaceRepoSlugPullrequestsParams{},
		withStates(states),
	)
	if err != nil {
		return nil, fmt.Errorf("error listing pull requests: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf(
			"unexpected response %d listing pull requests", resp.StatusCode(),
		)
	}

	// NB: The Bitbucket API doesn't support filtering by source/destination
	// branch or commit hash, so we have to filter client-side, which is
	// highly inefficient.
	var result []gitprovider.PullRequest
	if resp.JSON200.Values == nil {
		return result, nil
	}
	values := *resp.JSON200.Values
	for i := range values {
		pr := &values[i]
		if opts.HeadBranch != "" {
			branchName := ""
			if pr.Source != nil &&
				pr.Source.Branch != nil &&
				pr.Source.Branch.Name != nil {
				branchName = *pr.Source.Branch.Name
			}
			if branchName != opts.HeadBranch {
				continue
			}
		}
		if opts.BaseBranch != "" {
			branchName := ""
			if pr.Destination != nil &&
				pr.Destination.Branch != nil &&
				pr.Destination.Branch.Name != nil {
				branchName = *pr.Destination.Branch.Name
			}
			if branchName != opts.BaseBranch {
				continue
			}
		}
		if opts.HeadCommit != "" {
			commitHash := ""
			if pr.Source != nil &&
				pr.Source.Commit != nil &&
				pr.Source.Commit.Hash != nil {
				commitHash = *pr.Source.Commit.Hash
			}
			if commitHash != opts.HeadCommit {
				continue
			}
		}
		if err = p.resolveFullMergeCommitSHA(ctx, pr); err != nil {
			return nil, err
		}
		result = append(result, *toProviderPR(pr))
	}
	return result, nil
}

// MergePullRequest implements gitprovider.Interface.
func (p *provider) MergePullRequest(
	ctx context.Context,
	id int64,
	opts *gitprovider.MergePullRequestOpts,
) (*gitprovider.PullRequest, bool, error) {
	if opts == nil {
		opts = &gitprovider.MergePullRequestOpts{}
	}

	prID := int(id)

	getResp, err := p.client.GetRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdWithResponse(
		ctx,
		p.owner,
		p.repoSlug,
		prID,
	)
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if getResp.JSON200 == nil {
		return nil, false, fmt.Errorf(
			"unexpected response %d getting pull request %d", getResp.StatusCode(), id,
		)
	}
	pr := getResp.JSON200

	var state PullrequestState
	if pr.State != nil {
		state = *pr.State
	}

	if state == PullrequestStateMERGED {
		return toProviderPR(pr), true, nil
	}

	if state != PullrequestStateOPEN {
		return nil, false, fmt.Errorf(
			"pull request %d is closed but not merged (state: %s)", id, state,
		)
	}

	if pr.Draft != nil && *pr.Draft {
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

	var mergeStrategy *PullrequestMergeParametersMergeStrategy
	if opts.MergeMethod != "" {
		s := PullrequestMergeParametersMergeStrategy(opts.MergeMethod)
		if !s.Valid() {
			return nil, false, fmt.Errorf("unsupported merge strategy %q", opts.MergeMethod)
		}
		mergeStrategy = &s
	}

	mergeBody := PullrequestMergeParameters{
		Type:          "pullrequestMergeParameters",
		MergeStrategy: mergeStrategy,
	}

	mergeResp, err := p.client.PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeWithResponse(
		ctx,
		p.owner,
		p.repoSlug,
		prID,
		&PostRepositoriesWorkspaceRepoSlugPullrequestsPullRequestIdMergeParams{},
		mergeBody,
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}

	if mergeResp.StatusCode() == http.StatusAccepted {
		return nil, false, fmt.Errorf(
			"pull request %d merge was accepted asynchronously and cannot be awaited", id,
		)
	}

	if mergeResp.JSON200 == nil {
		return nil, false, fmt.Errorf(
			"unexpected response %d merging pull request %d", mergeResp.StatusCode(), id,
		)
	}

	mergedPR := mergeResp.JSON200
	var mergedState PullrequestState
	if mergedPR.State != nil {
		mergedState = *mergedPR.State
	}

	// Per the Bitbucket API docs, the merge endpoint returns 200 only on
	// success (409 for conflicts, 555 for timeout). A 200 response with a
	// non-merged state would be unexpected.
	if mergedState != PullrequestStateMERGED {
		return nil, false, fmt.Errorf(
			"unexpected state %q after merging pull request %d", mergedState, id,
		)
	}

	return toProviderPR(mergedPR), true, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(repoURL string, sha string) (string, error) {
	normalizedURL := urls.NormalizeGit(repoURL)
	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}
	return fmt.Sprintf("https://%s%s/commits/%s", parsedURL.Host, parsedURL.Path, sha), nil
}

// resolveFullMergeCommitSHA replaces the (possibly short) merge commit hash on
// pr with the full SHA. Bitbucket returns a short hash in the pull request
// response, so a separate commit lookup is required.
func (p *provider) resolveFullMergeCommitSHA(ctx context.Context, pr *Pullrequest) error {
	if pr.MergeCommit == nil || pr.MergeCommit.Hash == nil || *pr.MergeCommit.Hash == "" {
		return nil
	}
	resp, err := p.client.GetRepositoriesWorkspaceRepoSlugCommitCommitWithResponse(
		ctx,
		p.owner,
		p.repoSlug,
		*pr.MergeCommit.Hash,
	)
	if err != nil {
		return fmt.Errorf("error getting commit: %w", err)
	}
	if resp.JSON200 == nil || resp.JSON200.Hash == nil {
		return fmt.Errorf("unexpected response %d getting commit", resp.StatusCode())
	}
	pr.MergeCommit.Hash = resp.JSON200.Hash
	return nil
}

// toProviderPR converts a Pullrequest to a gitprovider.PullRequest.
func toProviderPR(pr *Pullrequest) *gitprovider.PullRequest {
	if pr == nil {
		return nil
	}

	var id int64
	if pr.Id != nil {
		id = int64(*pr.Id)
	}

	var state PullrequestState
	if pr.State != nil {
		state = *pr.State
	}

	var prURL string
	if pr.Links != nil && pr.Links.Html != nil && pr.Links.Html.Href != nil {
		prURL = *pr.Links.Html.Href
	}

	var headSHA string
	if pr.Source != nil && pr.Source.Commit != nil && pr.Source.Commit.Hash != nil {
		headSHA = *pr.Source.Commit.Hash
	}

	var mergeCommitSHA string
	if pr.MergeCommit != nil && pr.MergeCommit.Hash != nil {
		mergeCommitSHA = *pr.MergeCommit.Hash
	}

	return &gitprovider.PullRequest{
		Number: id,
		URL:    prURL,
		Open:   state == PullrequestStateOPEN,
		Merged: state == PullrequestStateMERGED,
		// NB: As a sign of true craftsmanship, or lack thereof, the Bitbucket
		// API returns a short commit SHA as merge commit hash. To get the full
		// commit SHA, we need to fetch the commit details separately.
		MergeCommitSHA: mergeCommitSHA,
		HeadSHA:        headSHA,
		CreatedAt:      pr.CreatedOn,
		Object:         pr,
	}
}

// withStates returns a RequestEditorFn that overrides the state query params.
// The Bitbucket API supports multiple state filters via repeated ?state= params,
// but the generated params struct only supports a single value.
func withStates(states []string) RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		q := req.URL.Query()
		q.Del("state")
		for _, s := range states {
			q.Add("state", s)
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
}

// parseRepoURL extracts host, owner and repo slug from a Bitbucket Cloud repository URL.
func parseRepoURL(repoURL string) (host, owner, slug string, err error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("parse Bitbucket Cloud URL %q: %w", repoURL, err)
	}
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid repository path in URL %q", u)
	}
	return u.Hostname(), parts[0], parts[1], nil
}
