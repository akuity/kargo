package datacenter

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	// ProviderName is the name used to register the Bitbucket Data Center provider.
	ProviderName = "bitbucket-datacenter"
)

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// We assume that any hostname containing "bitbucket" that is not
		// bitbucket.org is a self-hosted Data Center instance. Instances that
		// don't include "bitbucket" in their hostname won't be auto-detected and
		// will require explicit provider configuration via opts.Name.
		host := u.Hostname()
		return strings.Contains(host, "bitbucket") && host != "bitbucket.org"
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

// provider is a Bitbucket Data Center implementation of gitprovider.Interface.
type provider struct {
	baseURL    string
	projectKey string
	repoSlug   string
	client     ClientWithResponsesInterface
}

// NewProvider returns a Bitbucket Data Center implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}

	baseURL, projectKey, repoSlug, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	token := opts.Token
	client, err := NewClientWithResponses(
		baseURL,
		WithHTTPClient(cleanhttp.DefaultClient()),
		WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Bitbucket Data Center API client: %w", err)
	}

	return &provider{
		baseURL:    baseURL,
		projectKey: projectKey,
		repoSlug:   repoSlug,
		client:     client,
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

	projectKey := p.projectKey
	repoSlug := p.repoSlug
	project := RestProject{Key: &projectKey}

	body := CreatePullRequestJSONRequestBody{
		Title: opts.Title,
		FromRef: RestCreateRef{
			Id: "refs/heads/" + opts.Head,
			Repository: RestRefRepository{
				Slug:    &repoSlug,
				Project: &project,
			},
		},
		ToRef: RestCreateRef{
			Id: "refs/heads/" + opts.Base,
			Repository: RestRefRepository{
				Slug:    &repoSlug,
				Project: &project,
			},
		},
	}
	if opts.Description != "" {
		body.Description = &opts.Description
	}

	resp, err := p.client.CreatePullRequestWithResponse(ctx, p.projectKey, p.repoSlug, body)
	if err != nil {
		return nil, fmt.Errorf("error creating pull request: %w", err)
	}
	if resp.JSON201 == nil {
		return nil, fmt.Errorf(
			"unexpected response %d creating pull request", resp.StatusCode(),
		)
	}
	return toProviderPR(resp.JSON201), nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	resp, err := p.client.GetPullRequestWithResponse(ctx, p.projectKey, p.repoSlug, int(id))
	if err != nil {
		return nil, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf(
			"unexpected response %d getting pull request %d", resp.StatusCode(), id,
		)
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

	var state GetPullRequestsParamsState
	switch opts.State {
	case gitprovider.PullRequestStateOpen:
		state = GetPullRequestsParamsStateOPEN
	case gitprovider.PullRequestStateClosed, gitprovider.PullRequestStateAny:
		state = GetPullRequestsParamsStateALL
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}

	resp, err := p.client.GetPullRequestsWithResponse(
		ctx,
		p.projectKey,
		p.repoSlug,
		&GetPullRequestsParams{State: &state},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing pull requests: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf(
			"unexpected response %d listing pull requests", resp.StatusCode(),
		)
	}

	var result []gitprovider.PullRequest
	if resp.JSON200.Values == nil {
		return result, nil
	}
	for i := range *resp.JSON200.Values {
		pr := &(*resp.JSON200.Values)[i]
		// When requesting ALL states for the closed filter, exclude open PRs.
		if opts.State == gitprovider.PullRequestStateClosed {
			var prState RestPullRequestState
			if pr.State != nil {
				prState = *pr.State
			}
			if prState == RestPullRequestStateOPEN {
				continue
			}
		}
		// NB: The Bitbucket Data Center API doesn't support filtering by
		// source/destination branch or commit hash, so we filter client-side.
		if opts.HeadBranch != "" {
			branchName := ""
			if pr.FromRef != nil && pr.FromRef.DisplayId != nil {
				branchName = *pr.FromRef.DisplayId
			}
			if branchName != opts.HeadBranch {
				continue
			}
		}
		if opts.BaseBranch != "" {
			branchName := ""
			if pr.ToRef != nil && pr.ToRef.DisplayId != nil {
				branchName = *pr.ToRef.DisplayId
			}
			if branchName != opts.BaseBranch {
				continue
			}
		}
		if opts.HeadCommit != "" {
			commitHash := ""
			if pr.FromRef != nil && pr.FromRef.LatestCommit != nil {
				commitHash = *pr.FromRef.LatestCommit
			}
			if commitHash != opts.HeadCommit {
				continue
			}
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

	getResp, err := p.client.GetPullRequestWithResponse(ctx, p.projectKey, p.repoSlug, prID)
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if getResp.JSON200 == nil {
		return nil, false, fmt.Errorf(
			"unexpected response %d getting pull request %d", getResp.StatusCode(), id,
		)
	}
	pr := getResp.JSON200

	var state RestPullRequestState
	if pr.State != nil {
		state = *pr.State
	}

	if state == RestPullRequestStateMERGED {
		return toProviderPR(pr), true, nil
	}

	if state != RestPullRequestStateOPEN {
		return nil, false, fmt.Errorf(
			"pull request %d is closed but not merged (state: %s)", id, state,
		)
	}

	if pr.Draft != nil && *pr.Draft {
		return nil, false, nil
	}

	var version int
	if pr.Version != nil {
		version = *pr.Version
	}

	var mergeStrategy *RestMergeStrategy
	if opts.MergeMethod != "" {
		s := RestMergeStrategyId(opts.MergeMethod)
		if !s.Valid() {
			return nil, false, fmt.Errorf("unsupported merge strategy %q", opts.MergeMethod)
		}
		mergeStrategy = &RestMergeStrategy{Id: &s}
	}

	mergeResp, err := p.client.MergePullRequestWithResponse(
		ctx,
		p.projectKey,
		p.repoSlug,
		prID,
		&MergePullRequestParams{Version: version},
		MergePullRequestJSONRequestBody{Strategy: mergeStrategy},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}
	if mergeResp.JSON200 == nil {
		return nil, false, fmt.Errorf(
			"unexpected response %d merging pull request %d", mergeResp.StatusCode(), id,
		)
	}

	mergedPR := mergeResp.JSON200
	var mergedState RestPullRequestState
	if mergedPR.State != nil {
		mergedState = *mergedPR.State
	}

	if mergedState != RestPullRequestStateMERGED {
		return nil, false, fmt.Errorf(
			"unexpected state %q after merging pull request %d", mergedState, id,
		)
	}

	return toProviderPR(mergedPR), true, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(_ string, sha string) (string, error) {
	var projectPath string
	if strings.HasPrefix(p.projectKey, "~") {
		projectPath = fmt.Sprintf(
			"/users/%s/repos/%s",
			strings.TrimPrefix(p.projectKey, "~"),
			p.repoSlug,
		)
	} else {
		projectPath = fmt.Sprintf("/projects/%s/repos/%s", p.projectKey, p.repoSlug)
	}
	return fmt.Sprintf("%s%s/commits/%s", p.baseURL, projectPath, sha), nil
}

// toProviderPR converts a RestPullRequest to a gitprovider.PullRequest.
func toProviderPR(pr *RestPullRequest) *gitprovider.PullRequest {
	if pr == nil {
		return nil
	}

	var id int64
	if pr.Id != nil {
		id = int64(*pr.Id)
	}

	var state RestPullRequestState
	if pr.State != nil {
		state = *pr.State
	}

	var prURL string
	if pr.Links != nil && pr.Links.Self != nil && len(*pr.Links.Self) > 0 {
		if href := (*pr.Links.Self)[0].Href; href != nil {
			prURL = *href
		}
	}

	var headSHA string
	if pr.FromRef != nil && pr.FromRef.LatestCommit != nil {
		headSHA = *pr.FromRef.LatestCommit
	}

	var createdAt *time.Time
	if pr.CreatedDate != nil {
		t := time.UnixMilli(*pr.CreatedDate).UTC()
		createdAt = &t
	}

	return &gitprovider.PullRequest{
		Number:    id,
		URL:       prURL,
		Open:      state == RestPullRequestStateOPEN,
		Merged:    state == RestPullRequestStateMERGED,
		HeadSHA:   headSHA,
		CreatedAt: createdAt,
		Object:    pr,
	}
}

// parseRepoURL extracts the API base URL, project key, and repo slug from a
// Bitbucket Data Center repository URL. It handles three formats:
//
//   - Web UI:    https://host/projects/{key}/repos/{slug}
//     https://host/users/{username}/repos/{slug}
//   - HTTP clone: https://host/scm/{key}/{slug}
//   - SSH clone:  ssh://host/{key}/{slug}  (after NormalizeGit)
func parseRepoURL(repoURL string) (baseURL, projectKey, repoSlug string, err error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", fmt.Errorf("parse Bitbucket Data Center URL %q: %w", repoURL, err)
	}

	host := u.Hostname()
	if port := u.Port(); port != "" {
		host = host + ":" + port
	}
	baseURL = "https://" + host

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")

	switch {
	case len(parts) == 4 && parts[2] == "repos" && (parts[0] == "projects" || parts[0] == "users"):
		// Web UI URL: /projects/{key}/repos/{slug} or /users/{username}/repos/{slug}
		if parts[0] == "users" {
			projectKey = "~" + parts[1]
		} else {
			projectKey = parts[1]
		}
		repoSlug = parts[3]
	case len(parts) == 3 && parts[0] == "scm":
		// HTTP clone URL: /scm/{key}/{slug}
		projectKey = parts[1]
		repoSlug = parts[2]
	case len(parts) == 2:
		// SSH clone URL (after NormalizeGit): /{key}/{slug} or /~{username}/{slug}
		projectKey = parts[0]
		repoSlug = parts[1]
	default:
		return "", "", "", fmt.Errorf(
			"invalid repository path in URL %q: expected /projects/{key}/repos/{slug}, /scm/{key}/{slug}, or /{key}/{slug}",
			repoURL,
		)
	}
	return baseURL, projectKey, repoSlug, nil
}
