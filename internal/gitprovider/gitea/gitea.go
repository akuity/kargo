package gitea

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/gitprovider"
)

const ProviderName = "gitea"

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
		ctx context.Context,
		owner string,
		repo string,
		opts *gitea.CreatePullRequestOption,
	) (*gitea.PullRequest, *gitea.Response, error)

	ListPullRequests(
		ctx context.Context,
		owner string,
		repo string,
		opts *gitea.ListPullRequestsOptions,
	) ([]*gitea.PullRequest, *gitea.Response, error)

	GetPullRequests(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) (*gitea.PullRequest, *gitea.Response, error)

	AddLabelsToIssue(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		labels []string,
	) ([]*gitea.Label, *gitea.Response, error)
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
		client: &giteaClientWrapper{client},
	}, nil
}

type giteaClientWrapper struct {
	client *gitea.Client
}

func (g giteaClientWrapper) CreatePullRequest(
	_ context.Context,
	owner string,
	repo string,
	opts *gitea.CreatePullRequestOption,
) (*gitea.PullRequest, *gitea.Response, error) {
	return g.client.CreatePullRequest(owner, repo, *opts)
}

func (g giteaClientWrapper) ListPullRequests(
	_ context.Context,
	owner string,
	repo string,
	opts *gitea.ListPullRequestsOptions,
) ([]*gitea.PullRequest, *gitea.Response, error) {
	return g.client.ListRepoPullRequests(owner, repo, *opts)
}

func (g giteaClientWrapper) GetPullRequests(
	_ context.Context,
	owner string,
	repo string,
	number int,
) (*gitea.PullRequest, *gitea.Response, error) {
	return g.client.GetPullRequest(owner, repo, int64(number))
}

func (g giteaClientWrapper) AddLabelsToIssue(
	_ context.Context,
	owner string,
	repo string,
	number int,
	_ []string,
) ([]*gitea.Label, *gitea.Response, error) {
	return g.client.AddIssueLabels(owner, repo, int64(number), gitea.IssueLabelsOption{})
}

// CreatePullRequest implements gitprovider.Interface.
func (p *provider) CreatePullRequest(
	ctx context.Context,
	opts *gitprovider.CreatePullRequestOpts,
) (*gitprovider.PullRequest, error) {
	if opts == nil {
		opts = &gitprovider.CreatePullRequestOpts{}
	}
	giteaPR, _, err := p.client.CreatePullRequest(ctx,
		p.owner,
		p.repo,
		&gitea.CreatePullRequestOption{
			Title: opts.Title,
			Head:  opts.Head,
			Base:  opts.Base,
			Body:  opts.Description,
		},
	)
	if err != nil {
		return nil, err
	}
	if giteaPR == nil {
		return nil, fmt.Errorf("unexpected nil pull request")
	}
	pr := convertGiteaPR(*giteaPR)
	if len(opts.Labels) > 0 {
		if _, _, err = p.client.AddLabelsToIssue(ctx,
			p.owner,
			p.repo,
			int(pr.Number),
			opts.Labels,
		); err != nil {
			return nil, err
		}
	}
	return &pr, nil
}

// GetPullRequest implements gitprovider.Interface.
func (p *provider) GetPullRequest(
	ctx context.Context,
	id int64,
) (*gitprovider.PullRequest, error) {
	ghPR, _, err := p.client.GetPullRequests(ctx, p.owner, p.repo, int(id))
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
	ctx context.Context,
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
		giteaPRs, res, err := p.client.ListPullRequests(ctx, p.owner, p.repo, &listOpts)
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

func convertGiteaPR(giteaPR gitea.PullRequest) gitprovider.PullRequest {
	pr := gitprovider.PullRequest{
		Number:  giteaPR.Index,
		URL:     giteaPR.URL,
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
	u, err := url.Parse(git.NormalizeURL(repoURL))
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
