package governance

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/jferrl/go-githubauth"
	"golang.org/x/oauth2"
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
// methods needed by the governance bot.
// *github.PullRequestsService satisfies this interface.
type PullRequestsClient interface {
	Edit(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		pull *github.PullRequest,
	) (*github.PullRequest, *github.Response, error)
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

// githubClientFactory is an implementation of GitHubClientFactory that creates
// GitHub clients authenticated as a GitHub App installation. It caches clients
// for each installation ID to avoid creating multiple clients for the same
// installation.
type githubClientFactory struct {
	appTokenSource oauth2.TokenSource

	clientsMu sync.Mutex
	clients   map[int64]*github.Client

	newClientFn func(installationID int64) (*github.Client, error)
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
	return client.Issues, nil
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
	return client.PullRequests, nil
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
	return client.Repositories, nil
}

// getOrCreateClient retrieves a cached GitHub client for the given installation
// ID or creates a new one if it doesn't exist. It uses the appTokenSource to
// obtain an installation access token and creates a new GitHub client
// authenticated with that token. The client is cached for future use.
func (f *githubClientFactory) getOrCreateClient(
	installationID int64,
) (*github.Client, error) {
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
		f.clients = make(map[int64]*github.Client)
	}
	f.clients[installationID] = client
	return client, nil
}

func (f *githubClientFactory) newClient(
	installationID int64,
) (*github.Client, error) {
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
	return github.NewClient(cleanhttp.DefaultClient()).
		WithAuthToken(token.AccessToken), nil
}
