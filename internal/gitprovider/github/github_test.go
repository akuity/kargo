package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/gitprovider"
)

const testRepoOwner = "akuity"
const testRepoName = "kargo"

func TestParseGitHubURL(t *testing.T) {
	testCases := []struct {
		url            string
		expectedScheme string
		expectedHost   string
		expectedOwner  string
		expectedRepo   string
		errExpected    bool
	}{
		{
			url:         "not-a-url",
			errExpected: true,
		},
		{
			url:         "https://github.com/akuity",
			errExpected: true,
		},
		{
			url:            "https://github.com/akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			url:            "https://github.com/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			// This isn't a real URL. It's just to validate that the function can
			// handle GitHub Enterprise URLs.
			url:            "https://github.akuity.io/akuity/kargo.git",
			expectedScheme: "https",
			expectedHost:   "github.akuity.io",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			url:            "http://git@example.com:8080/akuity/kargo",
			errExpected:    false,
			expectedScheme: "http",
			expectedHost:   "example.com:8080",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
		{
			url:            "git@github.com:akuity/kargo",
			expectedScheme: "https",
			expectedHost:   "github.com",
			expectedOwner:  "akuity",
			expectedRepo:   "kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.url, func(t *testing.T) {
			scheme, host, owner, repo, err := parseRepoURL(testCase.url)
			if testCase.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testCase.expectedScheme, scheme)
				require.Equal(t, testCase.expectedHost, host)
				require.Equal(t, testCase.expectedOwner, owner)
				require.Equal(t, testCase.expectedRepo, repo)
			}
		})
	}
}

type mockGithubClient struct {
	mock.Mock
	pr       *github.PullRequest
	owner    string
	repo     string
	newPr    *github.NewPullRequest
	labels   []string
	listOpts *github.PullRequestListOptions
}

func (m *mockGithubClient) ListPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	m.owner = owner
	m.repo = repo
	m.listOpts = opts
	prs, ok := args.Get(0).([]*github.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return prs, nil, args.Error(2)
	}
	return prs, resp, args.Error(2)
}

func (m *mockGithubClient) GetPullRequests(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number)
	m.owner = owner
	m.repo = repo
	pr, ok := args.Get(0).(*github.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return pr, nil, args.Error(2)
	}
	return pr, resp, args.Error(2)
}

func (m *mockGithubClient) AddLabelsToIssue(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	labels []string,
) ([]*github.Label, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, labels)
	m.labels = labels
	labelsResp, ok := args.Get(0).([]*github.Label)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return labelsResp, nil, args.Error(2)
	}
	return labelsResp, resp, args.Error(2)
}
func (m *mockGithubClient) CreatePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	pull *github.NewPullRequest,
) (*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, pull)
	m.owner = owner
	m.repo = repo
	m.newPr = pull

	pr, ok := args.Get(0).(*github.PullRequest)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
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
	mockClient := &mockGithubClient{
		pr: &github.PullRequest{
			Number:         github.Int(42),
			MergeCommitSHA: github.String("sha"),
			State:          github.String("open"),
			URL:            github.String("url"),
		},
	}
	mockClient.
		On("CreatePullRequest", context.Background(), testRepoOwner, testRepoName, mock.Anything).
		Return(
			&github.PullRequest{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.String(opts.Head),
				},
				Base: &github.PullRequestBranch{
					Ref: github.String(opts.Base),
				},
				Title:          github.String(opts.Title),
				Body:           github.String(opts.Description),
				MergeCommitSHA: mockClient.pr.MergeCommitSHA,
				State:          mockClient.pr.State,
				HTMLURL:        mockClient.pr.URL,
			},
			&github.Response{},
			nil,
		)
	mockClient.
		On("AddLabelsToIssue", context.Background(), testRepoOwner, testRepoName, *mockClient.pr.Number, mock.Anything).
		Return(
			[]*github.Label{},
			&github.Response{},
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
	require.Equal(t, opts.Head, *mockClient.newPr.Head)
	require.Equal(t, opts.Base, *mockClient.newPr.Base)
	require.Equal(t, opts.Title, *mockClient.newPr.Title,
		"Expected title in new PR request to match title from options")
	require.Equal(t, opts.Description, *mockClient.newPr.Body,
		"Expected body in new PR request to match description from options")
	require.ElementsMatch(t, opts.Labels, mockClient.labels,
		"Expected labels passed to GitHub client to match labels from options")

	require.Equal(t, int64(*mockClient.pr.Number), pr.Number,
		"Expected PR number in returned object to match what was returned by GitHub")
	require.Equal(t, *mockClient.pr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, *mockClient.pr.URL, pr.URL)
	require.True(t, pr.Open)
}

func TestGetPullRequest(t *testing.T) {
	// set up mock
	mockClient := &mockGithubClient{
		pr: &github.PullRequest{
			Number:         github.Int(42),
			MergeCommitSHA: github.String("sha"),
			State:          github.String("open"),
			URL:            github.String("url"),
		},
	}
	mockClient.
		On("GetPullRequests", context.Background(), testRepoOwner, testRepoName, *mockClient.pr.Number).
		Return(
			&github.PullRequest{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.String("head"),
				},
				MergeCommitSHA: mockClient.pr.MergeCommitSHA,
				State:          mockClient.pr.State,
				HTMLURL:        mockClient.pr.URL,
			},
			&github.Response{},
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
	require.Equal(t, int64(*mockClient.pr.Number), pr.Number,
		"Expected PR number in returned object to match what was returned by GitHub")
	require.Equal(t, *mockClient.pr.MergeCommitSHA, pr.MergeCommitSHA)
	require.Equal(t, *mockClient.pr.URL, pr.URL)
	require.True(t, pr.Open)
}

func TestListPullRequests(t *testing.T) {
	opts := gitprovider.ListPullRequestOptions{
		State:      gitprovider.PullRequestStateAny,
		HeadBranch: "head",
		BaseBranch: "base",
	}

	// set up mock
	mockClient := &mockGithubClient{
		pr: &github.PullRequest{
			Number:         github.Int(42),
			MergeCommitSHA: github.String("sha"),
			State:          github.String("open"),
			URL:            github.String("url"),
		},
	}
	mockClient.
		On("ListPullRequests", context.Background(), testRepoOwner, testRepoName, &github.PullRequestListOptions{
			State:     "all",
			Head:      opts.HeadBranch,
			Base:      opts.BaseBranch,
			Sort:      "",
			Direction: "",
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: 100,
			},
		}).
		Return(
			[]*github.PullRequest{{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.String("head"),
				},
				MergeCommitSHA: mockClient.pr.MergeCommitSHA,
				State:          mockClient.pr.State,
				HTMLURL:        mockClient.pr.URL,
			}},
			&github.Response{},
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
	require.Equal(t, opts.HeadBranch, mockClient.listOpts.Head)
	require.Equal(t, opts.BaseBranch, mockClient.listOpts.Base)

	require.Equal(t, int64(*mockClient.pr.Number), prs[0].Number)
	require.Equal(t, *mockClient.pr.MergeCommitSHA, prs[0].MergeCommitSHA)
	require.Equal(t, *mockClient.pr.URL, prs[0].URL)
	require.True(t, prs[0].Open)
}

func TestGetCommitURL(t *testing.T) {

	testCases := []struct {
		url         string
		sha         string
		expectedURL string
	}{
		{
			url:         "ssh://git@github.com:akuity/kargo.git",
			sha:         "sha",
			expectedURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			url:         "git@github.com:akuity/kargo.git",
			sha:         "sha",
			expectedURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			url:         "https://username@github.com/akuity/kargo",
			sha:         "sha",
			expectedURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			url:         "http://github.com/akuity/kargo.git",
			sha:         "sha",
			expectedURL: "https://github.com/akuity/kargo/commit/sha",
		},
	}

	for _, testCase := range testCases {
		// call the code we are testing
		g := provider{}
		commitURL, err := g.GetCommitURL(testCase.url, testCase.sha)
		require.NoError(t, err)
		require.Equal(t, testCase.expectedURL, commitURL)
	}
}
