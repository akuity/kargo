package gitea

import (
	"context"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/gitprovider"
)

const testRepoOwner = "akuity"
const testRepoName = "kargo"

func TestParsegiteaURL(t *testing.T) {
	testCases := []struct {
		url           string
		expectedHost  string
		expectedOwner string
		expectedRepo  string
		errExpected   bool
	}{
		{
			url:         "not-a-url",
			errExpected: true,
		},
		{
			url:         "https://git.domain.com/akuity",
			errExpected: true,
		},
		{
			url:           "https://git.domain.com/akuity/kargo",
			expectedHost:  "git.domain.com",
			expectedOwner: "akuity",
			expectedRepo:  "kargo",
		},
		{
			url:           "https://git.domain.com/akuity/kargo.git",
			expectedHost:  "git.domain.com",
			expectedOwner: "akuity",
			expectedRepo:  "kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.url, func(t *testing.T) {
			host, owner, repo, err := parseRepoURL(testCase.url)
			if testCase.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedHost, host)
				require.Equal(t, testCase.expectedOwner, owner)
				require.Equal(t, testCase.expectedRepo, repo)
			}
		})
	}
}

type mockGiteaClient struct {
	mock.Mock
	pr       *gitea.PullRequest
	owner    string
	repo     string
	newPr    *gitea.PullRequest
	labels   []string
	listOpts *gitea.ListPullRequestsOptions
}

func (m *mockGiteaClient) ListPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	opts *gitea.ListPullRequestsOptions,
) ([]*gitea.PullRequest, *gitea.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	m.owner = owner
	m.repo = repo
	m.listOpts = opts
	prs, ok := args.Get(0).([]*gitea.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return prs, nil, args.Error(2)
	}
	return prs, resp, args.Error(2)
}

func (m *mockGiteaClient) GetPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*gitea.PullRequest, *gitea.Response, error) {
	args := m.Called(ctx, owner, repo, number)
	m.owner = owner
	m.repo = repo
	pr, ok := args.Get(0).(*gitea.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func (m *mockGiteaClient) AddLabelsToIssue(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	labels []string,
) ([]*gitea.Label, *gitea.Response, error) {
	args := m.Called(ctx, owner, repo, number, labels)
	m.labels = labels
	labelsResp, ok := args.Get(0).([]*gitea.Label)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return labelsResp, nil, args.Error(2)
	}
	return labelsResp, resp, args.Error(2)
}

func (m *mockGiteaClient) CreatePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	opts *gitea.CreatePullRequestOption,
) (*gitea.PullRequest, *gitea.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	m.owner = owner
	m.repo = repo

	pr, ok := args.Get(0).(*gitea.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*gitea.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func TestCreatePullRequestWithLabels(t *testing.T) {
	opts := gitprovider.CreatePullRequestOpts{
		Head:        "feature-branch",
		Base:        "main",
		Title:       "title",
		Description: "desc",
		Labels:      []string{"label1", "label2"},
	}

	// set up mock
	mockClient := &mockGiteaClient{
		pr: &gitea.PullRequest{
			ID: *gitea.OptionalInt64(42),
		},
	}
	mockClient.
		On("CreatePullRequest", context.Background(), testRepoOwner, testRepoName, mock.Anything).
		Return(
			&gitea.PullRequest{
				State:   mockClient.pr.State,
				HTMLURL: mockClient.pr.URL,
			},
			&gitea.Response{},
			nil,
		)
	mockClient.
		On("AddLabelsToIssue", context.Background(), testRepoOwner, testRepoName, mockClient.pr.ID, mock.Anything).
		Return(
			[]*gitea.Label{},
			&gitea.Response{},
			nil,
		)

	// call the code we are testing
	g := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}
	pr, err := g.CreatePullRequest(context.Background(), &opts)

	// assert that the expectations were met
	mockClient.AssertExpectations(t)

	// other assertions
	require.NoError(t, err)
	require.Equal(t, testRepoOwner, mockClient.owner)
	require.Equal(t, testRepoName, mockClient.repo)
	require.Equal(t, opts.Head, mockClient.newPr.Head)
	require.Equal(t, opts.Base, mockClient.newPr.Base)
	require.Equal(t, opts.Title, mockClient.newPr.Title,
		"Expected title in new PR request to match title from options")
	require.Equal(t, opts.Description, mockClient.newPr.Body,
		"Expected body in new PR request to match description from options")
	require.ElementsMatch(t, opts.Labels, mockClient.labels,
		"Expected labels passed to gitea client to match labels from options")

	require.Equal(t, mockClient.pr.ID, pr.Number,
		"Expected PR number in returned object to match what was returned by gitea")
	require.Equal(t, mockClient.pr.Base.Sha, pr.MergeCommitSHA)
	require.Equal(t, mockClient.pr.URL, pr.URL)
	require.True(t, pr.Open)
}

func TestGetPullRequest(t *testing.T) {
	// set up mock
	mockClient := &mockGiteaClient{
		pr: &gitea.PullRequest{
			ID:  *gitea.OptionalInt64(42),
			URL: *gitea.OptionalString("http://localhost:8080"),
		},
	}

	mockClient.
		On("GetPullRequests", context.Background(), testRepoOwner, testRepoName, mockClient.pr.ID).
		Return(
			&gitea.PullRequest{
				ID:      mockClient.pr.ID,
				State:   mockClient.pr.State,
				HTMLURL: mockClient.pr.URL,
			},
			&gitea.Response{},
			nil,
		)

	// call the code we are testing
	g := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}
	pr, err := g.GetPullRequest(context.Background(), 42)

	// assert that the expectations were met
	mockClient.AssertExpectations(t)

	// other assertions
	require.NoError(t, err)
	require.Equal(t, testRepoOwner, mockClient.owner)
	require.Equal(t, testRepoName, mockClient.repo)
	require.Equal(t, mockClient.pr.ID, pr.Number,
		"Expected PR number in returned object to match what was returned by gitea")
	require.Equal(t, mockClient.pr.Base.Sha, pr.MergeCommitSHA)
	require.Equal(t, mockClient.pr.URL, pr.URL)
	require.True(t, pr.Open)
}

func TestListPullRequests(t *testing.T) {
	opts := gitprovider.ListPullRequestOptions{
		State:      gitprovider.PullRequestStateAny,
		HeadBranch: "head",
		BaseBranch: "base",
	}

	// set up mock
	mockClient := &mockGiteaClient{
		pr: &gitea.PullRequest{
			ID: *gitea.OptionalInt64(42),
		},
	}
	mockClient.
		On("ListPullRequests", context.Background(), testRepoOwner, testRepoName, &gitea.ListPullRequestsOptions{
			State: "all",
			ListOptions: gitea.ListOptions{
				Page:     0,
				PageSize: 100,
			},
		}).
		Return(
			[]*gitea.PullRequest{{
				ID: mockClient.pr.ID,
			}},
			&gitea.Response{},
			nil,
		)

	// call the code we are testing
	g := provider{
		owner:  testRepoOwner,
		repo:   testRepoName,
		client: mockClient,
	}

	prs, err := g.ListPullRequests(context.Background(), &opts)
	require.NoError(t, err)

	require.Equal(t, testRepoOwner, mockClient.owner)
	require.Equal(t, testRepoName, mockClient.repo)

	require.Equal(t, mockClient.pr.ID, prs[0].Number)
	require.Equal(t, mockClient.pr.Base.Sha, prs[0].MergeCommitSHA)
	require.Equal(t, mockClient.pr.URL, prs[0].URL)
	require.True(t, prs[0].Open)
}
