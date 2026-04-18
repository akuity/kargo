package governance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/jferrl/go-githubauth"
	"golang.org/x/oauth2"
)

const (
	// defaultGraphQLURL is the public GitHub GraphQL endpoint.
	defaultGraphQLURL = "https://api.github.com/graphql"

	// graphQLResponseMaxBytes caps the size of a GraphQL response body read into
	// memory. Successful convertPullRequestToDraft responses are tiny; anything
	// larger is either a GitHub error payload or suspicious.
	graphQLResponseMaxBytes = 1 << 20
)

// IssuesClient is the subset of github.IssuesService methods needed by
// the governance bot. *github.IssuesService satisfies this interface.
type IssuesClient interface {
	AddAssignees(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		assignees []string,
	) (*github.Issue, *github.Response, error)
	AddLabelsToIssue(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		labels []string,
	) ([]*github.Label, *github.Response, error)
	CreateComment(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		comment *github.IssueComment,
	) (*github.IssueComment, *github.Response, error)
	Edit(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		issue *github.IssueRequest,
	) (*github.Issue, *github.Response, error)
	Get(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) (*github.Issue, *github.Response, error)
	RemoveLabelForIssue(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		label string,
	) (*github.Response, error)
}

// PullRequestsClient is the subset of github.PullRequestsService
// methods needed by the governance bot, plus ConvertToDraft which is
// implemented via the GraphQL API (REST has no equivalent).
type PullRequestsClient interface {
	Edit(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		pull *github.PullRequest,
	) (*github.PullRequest, *github.Response, error)
	Get(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) (*github.PullRequest, *github.Response, error)
	ConvertToDraft(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) error
}

// RepositoriesClient is the subset of github.RepositoriesService
// methods needed by the governance bot.
// *github.RepositoriesService satisfies this interface.
type RepositoriesClient interface {
	GetContents(
		ctx context.Context,
		owner string,
		repo string,
		path string,
		opts *github.RepositoryContentGetOptions,
	) (
		*github.RepositoryContent,
		[]*github.RepositoryContent,
		*github.Response,
		error,
	)
}

// GitHubClientFactory creates authenticated GitHub service clients for
// a given installation.
type GitHubClientFactory interface {
	// NewIssuesClient creates a new IssuesClient authenticated as the GitHub App
	// installation with the specified installation ID.
	NewIssuesClient(installationID int64) (IssuesClient, error)
	// NewPullRequestsClient creates a new PullRequestsClient authenticated as the
	// GitHub App installation with the specified installation ID.
	NewPullRequestsClient(installationID int64) (PullRequestsClient, error)
	// NewRepositoriesClient creates a new RepositoriesClient authenticated as the
	// GitHub App installation with the specified installation ID.
	NewRepositoriesClient(installationID int64) (RepositoriesClient, error)
}

// installationClient bundles a per-installation *github.Client with its
// installation access token. The token is held so we can issue GraphQL
// requests for capabilities the REST API does not expose (e.g. converting a
// PR to draft).
type installationClient struct {
	client      *github.Client
	accessToken string
}

// githubClientFactory is an implementation of GitHubClientFactory that creates
// GitHub clients authenticated as a GitHub App installation. It caches clients
// for each installation ID to avoid creating multiple clients for the same
// installation.
type githubClientFactory struct {
	appTokenSource oauth2.TokenSource

	clientsMu sync.Mutex
	clients   map[int64]*installationClient

	newClientFn func(installationID int64) (*installationClient, error)
}

// NewGitHubClientFactory returns a GitHubClientFactory that
// authenticates as a GitHub App installation.
func NewGitHubClientFactory(
	clientID string,
	privateKey []byte,
) (GitHubClientFactory, error) {
	appTokenSource, err := githubauth.NewApplicationTokenSource(
		clientID,
		privateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating application token source: %w", err)
	}
	f := &githubClientFactory{appTokenSource: appTokenSource}
	f.newClientFn = f.newClient
	return f, nil
}

// NewIssuesClient implements GitHubClientFactory.
func (f *githubClientFactory) NewIssuesClient(
	installationID int64,
) (IssuesClient, error) {
	client, err := f.getOrCreateClient(installationID)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating GitHub issues service client for installation %d: %w",
			installationID, err,
		)
	}
	return client.client.Issues, nil
}

// NewPullRequestsClient implements GitHubClientFactory.
func (f *githubClientFactory) NewPullRequestsClient(
	installationID int64,
) (PullRequestsClient, error) {
	client, err := f.getOrCreateClient(installationID)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating GitHub pull requests service client for installation %d: %w",
			installationID, err,
		)
	}
	return &pullRequestsClient{
		PullRequestsService: client.client.PullRequests,
		accessToken:         client.accessToken,
	}, nil
}

// pullRequestsClient wraps *github.PullRequestsService to add the
// ConvertToDraft capability, which GitHub's REST API does not expose. The
// mutation is issued against the GraphQL endpoint using the installation's
// access token.
type pullRequestsClient struct {
	*github.PullRequestsService
	accessToken string
	// graphQLURL and httpClient are overridable for tests. Zero values use the
	// public GitHub GraphQL endpoint and cleanhttp's default client.
	graphQLURL string
	httpClient *http.Client
}

// ConvertToDraft implements PullRequestsClient. It resolves the PR's node ID
// via REST, then issues the convertPullRequestToDraft GraphQL mutation.
func (p *pullRequestsClient) ConvertToDraft(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) error {
	pr, _, err := p.Get(ctx, owner, repo, number)
	if err != nil {
		return fmt.Errorf("error fetching PR: %w", err)
	}
	nodeID := pr.GetNodeID()
	if nodeID == "" {
		return fmt.Errorf("PR #%d has no node ID", number)
	}

	body, err := json.Marshal(map[string]any{
		"query": `mutation($id: ID!) {
			convertPullRequestToDraft(input: {pullRequestId: $id}) {
				pullRequest { isDraft }
			}
		}`,
		"variables": map[string]string{"id": nodeID},
	})
	if err != nil {
		return fmt.Errorf("error marshaling GraphQL request: %w", err)
	}

	url := p.graphQLURL
	if url == "" {
		url = defaultGraphQLURL
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, url, bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("error building GraphQL request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpClient := p.httpClient
	if httpClient == nil {
		httpClient = cleanhttp.DefaultClient()
	}
	// URL is the hardcoded GitHub GraphQL endpoint (or a test override set
	// within this package), not user input.
	resp, err := httpClient.Do(req) //nolint:gosec
	if err != nil {
		return fmt.Errorf("error posting GraphQL request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, graphQLResponseMaxBytes))
	if err != nil {
		return fmt.Errorf("error reading GraphQL response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"GraphQL request failed with status %d: %s",
			resp.StatusCode, string(respBody),
		)
	}

	var parsed struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("error parsing GraphQL response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		msgs := make([]string, len(parsed.Errors))
		for i, e := range parsed.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("GraphQL errors: %s", strings.Join(msgs, "; "))
	}
	return nil
}

// NewRepositoriesClient implements GitHubClientFactory.
func (f *githubClientFactory) NewRepositoriesClient(
	installationID int64,
) (RepositoriesClient, error) {
	client, err := f.getOrCreateClient(installationID)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating GitHub repositories service client for installation %d: %w",
			installationID, err,
		)
	}
	return client.client.Repositories, nil
}

// getOrCreateClient retrieves a cached installationClient for the given
// installation ID or creates a new one if it doesn't exist.
func (f *githubClientFactory) getOrCreateClient(
	installationID int64,
) (*installationClient, error) {
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()
	if client, ok := f.clients[installationID]; ok {
		return client, nil
	}
	client, err := f.newClientFn(installationID)
	if err != nil {
		return nil, err
	}
	if f.clients == nil {
		f.clients = make(map[int64]*installationClient)
	}
	f.clients[installationID] = client
	return client, nil
}

func (f *githubClientFactory) newClient(
	installationID int64,
) (*installationClient, error) {
	installationTokenSource := githubauth.NewInstallationTokenSource(
		installationID,
		f.appTokenSource,
	)
	token, err := installationTokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf(
			"error getting installation access token: %w", err,
		)
	}
	return &installationClient{
		client: github.NewClient(cleanhttp.DefaultClient()).
			WithAuthToken(token.AccessToken),
		accessToken: token.AccessToken,
	}, nil
}
