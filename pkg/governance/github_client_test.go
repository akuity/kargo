package governance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

func Test_githubClientFactory_NewIssuesClient(t *testing.T) {
	testCases := []struct {
		name    string
		factory *githubClientFactory
		assert  func(*testing.T, IssuesClient, error)
	}{
		{
			name: "returns IssuesService from cached client",
			factory: func() *githubClientFactory {
				client := github.NewClient(nil)
				return &githubClientFactory{
					clients: map[int64]*installationClient{1: {client: client}},
				}
			}(),
			assert: func(t *testing.T, ic IssuesClient, err error) {
				require.NoError(t, err)
				require.NotNil(t, ic)
			},
		},
		{
			name: "wraps error with installation ID",
			factory: &githubClientFactory{
				newClientFn: func(_ int64) (*installationClient, error) {
					return nil, errors.New("boom")
				},
			},
			assert: func(t *testing.T, _ IssuesClient, err error) {
				require.ErrorContains(t, err, "installation 1")
				require.ErrorContains(t, err, "boom")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ic, err := testCase.factory.NewIssuesClient(1)
			testCase.assert(t, ic, err)
		})
	}
}

func Test_githubClientFactory_NewPullRequestsClient(t *testing.T) {
	testCases := []struct {
		name    string
		factory *githubClientFactory
		assert  func(*testing.T, PullRequestsClient, error)
	}{
		{
			name: "returns PullRequestsService from cached client",
			factory: func() *githubClientFactory {
				client := github.NewClient(nil)
				return &githubClientFactory{
					clients: map[int64]*installationClient{1: {client: client}},
				}
			}(),
			assert: func(t *testing.T, prc PullRequestsClient, err error) {
				require.NoError(t, err)
				require.NotNil(t, prc)
			},
		},
		{
			name: "wraps error with installation ID",
			factory: &githubClientFactory{
				newClientFn: func(_ int64) (*installationClient, error) {
					return nil, errors.New("boom")
				},
			},
			assert: func(t *testing.T, _ PullRequestsClient, err error) {
				require.ErrorContains(t, err, "installation 1")
				require.ErrorContains(t, err, "boom")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prc, err := testCase.factory.NewPullRequestsClient(1)
			testCase.assert(t, prc, err)
		})
	}
}

func Test_pullRequestsClient_ConvertToDraft(t *testing.T) {
	const (
		owner    = "akuity"
		repo     = "kargo"
		number   = 42
		nodeID   = "PR_kwDOabc"
		apiToken = "test-token"
	)

	testCases := []struct {
		name    string
		handler http.HandlerFunc
		assert  func(*testing.T, error)
	}{
		{
			// Earliest error path: fetching the PR via REST fails.
			name: "REST Get fails",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error fetching PR")
			},
		},
		{
			// State guard: closed PRs are a no-op. GraphQL must not be called.
			name: "closed PR is a no-op",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/graphql" {
					t.Fatalf("GraphQL should not be called for a closed PR")
				}
				_, _ = w.Write([]byte(
					`{"node_id": "` + nodeID + `", "state": "closed"}`,
				))
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Draft guard: already-draft PRs are a no-op. GraphQL must not be called.
			name: "already-draft PR is a no-op",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/graphql" {
					t.Fatalf("GraphQL should not be called for an already-draft PR")
				}
				_, _ = w.Write([]byte(
					`{"node_id": "` + nodeID + `", "state": "open", "draft": true}`,
				))
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			// Defensive: PR response missing node_id.
			name: "PR has no node ID",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t,
					"/repos/"+owner+"/"+repo+"/pulls/42", r.URL.Path,
				)
				_, _ = w.Write([]byte(`{}`))
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "no node ID")
			},
		},
		{
			// GraphQL transport error — HTTP status check branch.
			name: "GraphQL returns non-200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/" + owner + "/" + repo + "/pulls/42":
					_, _ = w.Write([]byte(`{"node_id": "` + nodeID + `"}`))
				case "/graphql":
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`unauthorized`))
				}
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "status 401")
			},
		},
		{
			// GraphQL response carries errors array — errors-field branch.
			name: "GraphQL returns error payload",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/" + owner + "/" + repo + "/pulls/42":
					_, _ = w.Write([]byte(`{"node_id": "` + nodeID + `"}`))
				case "/graphql":
					_, _ = w.Write([]byte(
						`{"errors":[{"message":"not authorized"}]}`,
					))
				}
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "GraphQL errors")
				require.ErrorContains(t, err, "not authorized")
			},
		},
		{
			// Success path.
			name: "happy path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer "+apiToken, r.Header.Get("Authorization"))
				switch {
				case r.Method == http.MethodGet &&
					r.URL.Path == "/repos/"+owner+"/"+repo+"/pulls/42":
					_, _ = w.Write([]byte(`{"node_id": "` + nodeID + `"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/graphql":
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					var payload struct {
						Variables map[string]string `json:"variables"`
					}
					require.NoError(t, json.Unmarshal(body, &payload))
					require.Equal(t, nodeID, payload.Variables["id"])
					_, _ = w.Write([]byte(
						`{"data":{"convertPullRequestToDraft":` +
							`{"pullRequest":{"isDraft":true}}}}`,
					))
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			srv := httptest.NewServer(testCase.handler)
			defer srv.Close()

			gh := github.NewClient(srv.Client()).WithAuthToken(apiToken)
			baseURL, err := url.Parse(srv.URL + "/")
			require.NoError(t, err)
			gh.BaseURL = baseURL

			p := &pullRequestsClient{
				PullRequestsService: gh.PullRequests,
				accessToken:         apiToken,
				graphQLURL:          srv.URL + "/graphql",
				httpClient:          srv.Client(),
			}

			err = p.ConvertToDraft(t.Context(), owner, repo, number)
			testCase.assert(t, err)
		})
	}
}

func Test_githubClientFactory_NewRepositoriesClient(t *testing.T) {
	testCases := []struct {
		name    string
		factory *githubClientFactory
		assert  func(*testing.T, RepositoriesClient, error)
	}{
		{
			name: "returns RepositoriesService from cached client",
			factory: func() *githubClientFactory {
				client := github.NewClient(nil)
				return &githubClientFactory{
					clients: map[int64]*installationClient{1: {client: client}},
				}
			}(),
			assert: func(t *testing.T, rc RepositoriesClient, err error) {
				require.NoError(t, err)
				require.NotNil(t, rc)
			},
		},
		{
			name: "wraps error with installation ID",
			factory: &githubClientFactory{
				newClientFn: func(_ int64) (*installationClient, error) {
					return nil, errors.New("boom")
				},
			},
			assert: func(t *testing.T, _ RepositoriesClient, err error) {
				require.ErrorContains(t, err, "installation 1")
				require.ErrorContains(t, err, "boom")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rc, err := testCase.factory.NewRepositoriesClient(1)
			testCase.assert(t, rc, err)
		})
	}
}

func Test_githubClientFactory_getOrCreateClient(t *testing.T) {
	testCases := []struct {
		name    string
		factory *githubClientFactory
		id      int64
		assert  func(*testing.T, *githubClientFactory, *installationClient, error)
	}{
		{
			name: "creates and caches a new client",
			factory: &githubClientFactory{
				newClientFn: func(_ int64) (*installationClient, error) {
					return &installationClient{client: github.NewClient(nil)}, nil
				},
			},
			id: 1,
			assert: func(
				t *testing.T,
				f *githubClientFactory,
				client *installationClient,
				err error,
			) {
				require.NoError(t, err)
				require.NotNil(t, client)
				require.Len(t, f.clients, 1)
				require.Same(t, client, f.clients[1])
			},
		},
		{
			name: "returns cached client on subsequent call",
			factory: func() *githubClientFactory {
				cached := &installationClient{client: github.NewClient(nil)}
				return &githubClientFactory{
					clients: map[int64]*installationClient{1: cached},
					newClientFn: func(_ int64) (*installationClient, error) {
						t.Fatal("newClientFn should not be called for cached client")
						return nil, nil
					},
				}
			}(),
			id: 1,
			assert: func(
				t *testing.T,
				f *githubClientFactory,
				client *installationClient,
				err error,
			) {
				require.NoError(t, err)
				require.Same(t, f.clients[1], client)
			},
		},
		{
			name: "caches separately per installation ID",
			factory: func() *githubClientFactory {
				clientA := &installationClient{client: github.NewClient(nil)}
				return &githubClientFactory{
					clients: map[int64]*installationClient{1: clientA},
					newClientFn: func(_ int64) (*installationClient, error) {
						return &installationClient{client: github.NewClient(nil)}, nil
					},
				}
			}(),
			id: 2,
			assert: func(
				t *testing.T,
				f *githubClientFactory,
				client *installationClient,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, f.clients, 2)
				require.NotSame(t, f.clients[1], f.clients[2])
				require.Same(t, f.clients[2], client)
			},
		},
		{
			name: "propagates error from newClientFn",
			factory: &githubClientFactory{
				newClientFn: func(_ int64) (*installationClient, error) {
					return nil, errors.New("token exchange failed")
				},
			},
			id: 1,
			assert: func(
				t *testing.T,
				f *githubClientFactory,
				_ *installationClient,
				err error,
			) {
				require.ErrorContains(t, err, "token exchange failed")
				require.Empty(t, f.clients)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, err := testCase.factory.getOrCreateClient(testCase.id)
			testCase.assert(t, testCase.factory, client, err)
		})
	}
}

// fakeClientFactory is a configurable fake GitHubClientFactory. Each New*
// method returns the corresponding fake client.
type fakeClientFactory struct {
	issuesClient IssuesClient
	prsClient    PullRequestsClient
	reposClient  RepositoriesClient
}

// NewIssuesClient implements GitHubClientFactory.
func (f *fakeClientFactory) NewIssuesClient(int64) (IssuesClient, error) {
	return f.issuesClient, nil
}

// NewPullRequestsClient implements GitHubClientFactory.
func (f *fakeClientFactory) NewPullRequestsClient(int64) (PullRequestsClient, error) {
	return f.prsClient, nil
}

// NewRepositoriesClient implements GitHubClientFactory.
func (f *fakeClientFactory) NewRepositoriesClient(int64) (RepositoriesClient, error) {
	return f.reposClient, nil
}

// fakeIssuesClient is a configurable fake IssuesClient. Each method
// delegates to a function field. Unconfigured methods are safe no-ops.
type fakeIssuesClient struct {
	AddAssigneesFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		assignees []string,
	) (*github.Issue, *github.Response, error)
	AddLabelsToIssueFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		labels []string,
	) ([]*github.Label, *github.Response, error)
	CreateCommentFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		comment *github.IssueComment,
	) (*github.IssueComment, *github.Response, error)
	EditFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		req *github.IssueRequest,
	) (*github.Issue, *github.Response, error)
	GetFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) (*github.Issue, *github.Response, error)
	RemoveLabelForIssueFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		label string,
	) (*github.Response, error)
}

// AddAssignees implements IssuesClient.
func (f *fakeIssuesClient) AddAssignees(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	assignees []string,
) (*github.Issue, *github.Response, error) {
	if f.AddAssigneesFn != nil {
		return f.AddAssigneesFn(ctx, owner, repo, number, assignees)
	}
	return nil, nil, nil
}

// AddLabelsToIssue implements IssuesClient.
func (f *fakeIssuesClient) AddLabelsToIssue(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	labels []string,
) ([]*github.Label, *github.Response, error) {
	if f.AddLabelsToIssueFn != nil {
		return f.AddLabelsToIssueFn(ctx, owner, repo, number, labels)
	}
	return nil, nil, nil
}

// CreateComment implements IssuesClient.
func (f *fakeIssuesClient) CreateComment(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	comment *github.IssueComment,
) (*github.IssueComment, *github.Response, error) {
	if f.CreateCommentFn != nil {
		return f.CreateCommentFn(ctx, owner, repo, number, comment)
	}
	return nil, nil, nil
}

// Edit implements IssuesClient.
func (f *fakeIssuesClient) Edit(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	req *github.IssueRequest,
) (*github.Issue, *github.Response, error) {
	if f.EditFn != nil {
		return f.EditFn(ctx, owner, repo, number, req)
	}
	return nil, nil, nil
}

// Get implements IssuesClient.
func (f *fakeIssuesClient) Get(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*github.Issue, *github.Response, error) {
	if f.GetFn != nil {
		return f.GetFn(ctx, owner, repo, number)
	}
	return &github.Issue{}, nil, nil
}

// RemoveLabelForIssue implements IssuesClient.
func (f *fakeIssuesClient) RemoveLabelForIssue(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	label string,
) (*github.Response, error) {
	if f.RemoveLabelForIssueFn != nil {
		return f.RemoveLabelForIssueFn(ctx, owner, repo, number, label)
	}
	return nil, nil
}

// fakePullRequestsClient is a configurable fake PullRequestsClient.
type fakePullRequestsClient struct {
	EditFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		pr *github.PullRequest,
	) (*github.PullRequest, *github.Response, error)
	GetFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) (*github.PullRequest, *github.Response, error)
	ListFilesFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
		opts *github.ListOptions,
	) ([]*github.CommitFile, *github.Response, error)
	ConvertToDraftFn func(
		ctx context.Context,
		owner string,
		repo string,
		number int,
	) error
}

// Edit implements PullRequestsClient.
func (f *fakePullRequestsClient) Edit(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	pr *github.PullRequest,
) (*github.PullRequest, *github.Response, error) {
	if f.EditFn != nil {
		return f.EditFn(ctx, owner, repo, number, pr)
	}
	return pr, nil, nil
}

// Get implements PullRequestsClient.
func (f *fakePullRequestsClient) Get(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*github.PullRequest, *github.Response, error) {
	if f.GetFn != nil {
		return f.GetFn(ctx, owner, repo, number)
	}
	return &github.PullRequest{}, nil, nil
}

// ListFiles implements PullRequestsClient.
func (f *fakePullRequestsClient) ListFiles(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	opts *github.ListOptions,
) ([]*github.CommitFile, *github.Response, error) {
	if f.ListFilesFn != nil {
		return f.ListFilesFn(ctx, owner, repo, number, opts)
	}
	return nil, nil, nil
}

// ConvertToDraft implements PullRequestsClient.
func (f *fakePullRequestsClient) ConvertToDraft(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) error {
	if f.ConvertToDraftFn != nil {
		return f.ConvertToDraftFn(ctx, owner, repo, number)
	}
	return nil
}

// fakeRepositoriesClient is a configurable fake RepositoriesClient.
// By default, GetContents marshals the config field as YAML and returns
// it as base64-encoded content.
type fakeRepositoriesClient struct {
	config        config
	GetContentsFn func(
		ctx context.Context,
		owner string,
		repo string,
		path string,
		opts *github.RepositoryContentGetOptions,
	) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
}

// GetContents implements RepositoriesClient.
func (f *fakeRepositoriesClient) GetContents(
	ctx context.Context,
	owner string,
	repo string,
	path string,
	opts *github.RepositoryContentGetOptions,
) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	if f.GetContentsFn != nil {
		return f.GetContentsFn(ctx, owner, repo, path, opts)
	}
	yamlBytes, err := yaml.Marshal(f.config)
	if err != nil {
		return nil, nil, nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(yamlBytes)
	return &github.RepositoryContent{
		Content:  github.Ptr(encoded),
		Encoding: github.Ptr("base64"),
	}, nil, nil, nil
}
