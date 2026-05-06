package gitea

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/urls"
)

const ProviderName = "gitea"

// validMergeMethods is the set of merge methods supported by Gitea's API. Gitea
// does not seem to validate this server-side, so we validate client-side.
var validMergeMethods = map[string]struct{}{
	"fast-forward-only": {},
	"manually-merged":   {},
	"merge":             {},
	"rebase":            {},
	"rebase-merge":      {},
	"squash":            {},
}

var registration = gitprovider.Registration{
	Predicate: func(repoURL string) bool {
		u, err := url.Parse(repoURL)
		if err != nil {
			return false
		}
		// We assume that any hostname with the word "gitea" in it can use this
		// provider
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

type giteaClient interface {
	CreatePullRequest(
		owner string,
		repo string,
		opts gitea.CreatePullRequestOption,
	) (*gitea.PullRequest, *gitea.Response, error)

	ListRepoPullRequests(
		owner string,
		repo string,
		opts gitea.ListPullRequestsOptions,
	) ([]*gitea.PullRequest, *gitea.Response, error)

	ListRepoLabels(
		owner string,
		repo string,
		opts gitea.ListLabelsOptions,
	) ([]*gitea.Label, *gitea.Response, error)

	GetPullRequest(
		owner string,
		repo string,
		number int64,
	) (*gitea.PullRequest, *gitea.Response, error)

	MergePullRequest(
		owner string,
		repo string,
		number int64,
		opts gitea.MergePullRequestOption,
	) (bool, *gitea.Response, error)
}

// provider is a Gitea implementation of gitprovider.Interface.
type provider struct { // nolint: revive
	owner  string
	repo   string
	client giteaClient
}

// NewProvider returns a Gitea-based implementation of gitprovider.Interface.
func NewProvider(
	repoURL string,
	opts *gitprovider.Options,
) (gitprovider.Interface, error) {
	if opts == nil {
		opts = &gitprovider.Options{}
	}

	scheme, host, owner, repo, err := parseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	var clientOpts []gitea.ClientOption
	if opts.Token != "" {
		clientOpts = append(clientOpts, gitea.SetToken(opts.Token))
	}

	httpClient := cleanhttp.DefaultClient()
	if opts.InsecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}
	clientOpts = append(clientOpts, gitea.SetHTTPClient(httpClient))

	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	client, err := gitea.NewClient(baseURL, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &provider{
		owner:  owner,
		repo:   repo,
		client: client,
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
	labelIDs, err := p.resolveLabelIDs(opts.Labels)
	if err != nil {
		return nil, err
	}
	giteaPR, _, err := p.client.CreatePullRequest(
		p.owner,
		p.repo,
		gitea.CreatePullRequestOption{
			Title:  opts.Title,
			Head:   opts.Head,
			Base:   opts.Base,
			Body:   opts.Description,
			Labels: labelIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	if giteaPR == nil {
		return nil, fmt.Errorf("unexpected nil pull request")
	}
	pr := convertGiteaPR(*giteaPR)
	return &pr, nil
}

func (p *provider) resolveLabelIDs(labelNames []string) ([]int64, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}

	labelIDsByName := make(map[string]int64, len(labelNames))
	for page := 1; ; {
		pageLabels, resp, err := p.client.ListRepoLabels(
			p.owner,
			p.repo,
			gitea.ListLabelsOptions{
				ListOptions: gitea.ListOptions{Page: page},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error listing repository labels: %w", err)
		}
		for _, repoLabel := range pageLabels {
			if repoLabel == nil {
				continue
			}
			labelIDsByName[repoLabel.Name] = repoLabel.ID
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	labelIDs := make([]int64, 0, len(labelNames))
	seen := make(map[int64]struct{}, len(labelNames))
	missing := make([]string, 0)
	for _, labelName := range labelNames {
		labelID, ok := labelIDsByName[labelName]
		if !ok {
			missing = append(missing, labelName)
			continue
		}
		if _, ok := seen[labelID]; ok {
			continue
		}
		seen[labelID] = struct{}{}
		labelIDs = append(labelIDs, labelID)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"labels not found in repository %s/%s: %s",
			p.owner,
			p.repo,
			strings.Join(missing, ", "),
		)
	}

	return labelIDs, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	_ context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	ghPR, _, err := p.client.GetPullRequest(p.owner, p.repo, id)
	if err != nil {
		return nil, err
	}
	if ghPR == nil {
		return nil, fmt.Errorf("unexpected nil pull request")
	}
	pr := convertGiteaPR(*ghPR)
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
	listOpts := gitea.ListPullRequestsOptions{
		ListOptions: gitea.ListOptions{},
	}
	switch opts.State {
	case gitprovider.PullRequestStateAny:
		listOpts.State = gitea.StateAll
	case gitprovider.PullRequestStateClosed:
		listOpts.State = gitea.StateClosed
	case gitprovider.PullRequestStateOpen:
		listOpts.State = gitea.StateOpen
	default:
		return nil, fmt.Errorf("unknown pull request state %q", opts.State)
	}
	var prs []gitprovider.PullRequest
	for {
		giteaPRs, res, err := p.client.ListRepoPullRequests(p.owner, p.repo, listOpts)
		if err != nil {
			return nil, err
		}
		for _, giteaPR := range giteaPRs {
			if opts.HeadCommit == "" || giteaPR.Head.Sha == opts.HeadCommit {
				prs = append(prs, convertGiteaPR(*giteaPR))
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
	opts *gitprovider.MergePullRequestOpts,
) (*gitprovider.PullRequest, bool, error) {
	if opts == nil {
		opts = &gitprovider.MergePullRequestOpts{}
	}

	giteaPR, _, err := p.client.GetPullRequest(p.owner, p.repo, id)
	if err != nil {
		return nil, false, fmt.Errorf("error getting pull request %d: %w", id, err)
	}
	if giteaPR == nil {
		return nil, false, fmt.Errorf("pull request %d not found", id)
	}

	switch {
	case giteaPR.HasMerged:
		pr := convertGiteaPR(*giteaPR)
		return &pr, true, nil

	case giteaPR.State != gitea.StateOpen:
		return nil, false, fmt.Errorf("pull request %d is closed but not merged", id)

	case giteaPR.Draft || !giteaPR.Mergeable:
		return nil, false, nil
	}

	mergeMethod := opts.MergeMethod
	if mergeMethod == "" {
		mergeMethod = "merge"
	}
	if _, ok := validMergeMethods[mergeMethod]; !ok {
		return nil, false,
			fmt.Errorf("unsupported merge method %q", mergeMethod)
	}

	merged, _, err := p.client.MergePullRequest(
		p.owner,
		p.repo,
		id,
		gitea.MergePullRequestOption{Style: gitea.MergeStyle(mergeMethod)},
	)
	if err != nil {
		return nil, false, fmt.Errorf("error merging pull request %d: %w", id, err)
	}
	if !merged {
		return nil, false, fmt.Errorf("merge rejected for pull request %d", id)
	}

	updatedPR, _, err := p.client.GetPullRequest(p.owner, p.repo, id)
	if err != nil {
		return nil, false, fmt.Errorf("error fetching PR %d after merge: %w", id, err)
	}
	if updatedPR == nil {
		return nil, false, fmt.Errorf("unexpected nil PR after merge")
	}

	pr := convertGiteaPR(*updatedPR)
	return &pr, true, nil
}

// GetCommitURL implements gitprovider.Interface.
func (p *provider) GetCommitURL(repoURL string, sha string) (string, error) {
	normalizedURL := urls.NormalizeGit(repoURL)

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("error processing repository URL: %s: %s", repoURL, err)
	}

	commitURL := fmt.Sprintf("https://%s%s/commit/%s", parsedURL.Host, parsedURL.Path, sha)

	return commitURL, nil
}

func convertGiteaPR(giteaPR gitea.PullRequest) gitprovider.PullRequest {
	pr := gitprovider.PullRequest{
		Number:  giteaPR.Index,
		URL:     giteaPR.HTMLURL,
		Open:    giteaPR.State == gitea.StateOpen,
		Merged:  giteaPR.HasMerged,
		Object:  giteaPR,
		HeadSHA: giteaPR.Head.Sha,
	}
	if giteaPR.MergedCommitID != nil {
		pr.MergeCommitSHA = *giteaPR.MergedCommitID
	}
	if giteaPR.Created != nil {
		pr.CreatedAt = giteaPR.Created
	}
	return pr
}

func parseRepoURL(repoURL string) (string, string, string, string, error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", "", fmt.Errorf(
			"error parsing gitea repository URL %q: %w", u, err,
		)
	}

	scheme := u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", "", fmt.Errorf(
			"could not extract repository owner and name from URL %q", u,
		)
	}

	return scheme, u.Host, parts[0], parts[1], nil
}
