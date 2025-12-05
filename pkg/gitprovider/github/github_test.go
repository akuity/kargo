package github

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
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

func (m *mockGithubClient) MergePullRequest(
	ctx context.Context,
	owner string,
	repo string,
	number int,
	commitMessage string,
	options *github.PullRequestOptions,
) (*github.PullRequestMergeResult, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, commitMessage, options)
	result, ok := args.Get(0).(*github.PullRequestMergeResult)
	if !ok {
		return nil, nil, args.Error(2)
	}
	resp, ok := args.Get(1).(*github.Response)
	if !ok {
		return result, nil, args.Error(2)
	}
	return result, resp, args.Error(2)
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
			Number:         github.Ptr(42),
			MergeCommitSHA: github.Ptr("sha"),
			State:          github.Ptr("open"),
			URL:            github.Ptr("url"),
		},
	}
	mockClient.
		On("CreatePullRequest", context.Background(), testRepoOwner, testRepoName, mock.Anything).
		Return(
			&github.PullRequest{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.Ptr(opts.Head),
				},
				Base: &github.PullRequestBranch{
					Ref: github.Ptr(opts.Base),
				},
				Title:          github.Ptr(opts.Title),
				Body:           github.Ptr(opts.Description),
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
			Number:         github.Ptr(42),
			MergeCommitSHA: github.Ptr("sha"),
			State:          github.Ptr("open"),
			URL:            github.Ptr("url"),
		},
	}
	mockClient.
		On("GetPullRequests", context.Background(), testRepoOwner, testRepoName, *mockClient.pr.Number).
		Return(
			&github.PullRequest{
				Number: mockClient.pr.Number,
				Head: &github.PullRequestBranch{
					Ref: github.Ptr("head"),
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
			Number:         github.Ptr(42),
			MergeCommitSHA: github.Ptr("sha"),
			State:          github.Ptr("open"),
			URL:            github.Ptr("url"),
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
					Ref: github.Ptr("head"),
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

func TestMergePullRequest(t *testing.T) {
	tests := []struct {
		name           string
		prNumber       int64
		setupMock      func(*mockGithubClient)
		expectedMerged bool
		expectError    bool
		errorContains  string
	}{
		{
			name:     "error getting initial PR state",
			prNumber: 999,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(999)).
					Return(nil, nil, errors.New("get PR failed"))
			},
			expectError:   true,
			errorContains: "error getting pull request",
		},
		{
			name:     "nil PR returned from initial get",
			prNumber: 404,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(404)).
					Return(nil, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "pull request 404 not found",
		},
		{
			name:     "PR already merged",
			prNumber: 123,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(123)).
					Return(&github.PullRequest{
						Number:         github.Ptr(123),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/123"),
						MergeCommitSHA: github.Ptr("merge_sha"),
						Head: &github.PullRequestBranch{
							SHA: github.Ptr("head_sha"),
						},
						MergedAt: &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil)
			},
			expectedMerged: true,
		},
		{
			name:     "PR closed but not merged",
			prNumber: 456,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(456)).
					Return(&github.PullRequest{
						Number:  github.Ptr(456),
						State:   github.Ptr("closed"),
						Merged:  github.Ptr(false),
						HTMLURL: github.Ptr("https://github.com/akuity/kargo/pull/456"),
						Head:    &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
					}, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "closed but not merged",
		},
		{
			name:     "unknown mergeability",
			prNumber: 444,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(444)).
					Return(&github.PullRequest{
						Number:    github.Ptr(444),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: nil,
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/444"),
					}, &github.Response{}, nil)
			},
			expectError: false,
		},
		{
			name:     "PR not ready to merge",
			prNumber: 789,
			setupMock: func(m *mockGithubClient) {
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(789)).
					Return(&github.PullRequest{
						Number:    github.Ptr(789),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(false),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/789"),
					}, &github.Response{}, nil)
			},
		},
		{
			name:     "merge call fails",
			prNumber: 555,
			setupMock: func(m *mockGithubClient) {
				// Get PR first
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(555)).
					Return(&github.PullRequest{
						Number:    github.Ptr(555),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/555"),
					}, &github.Response{}, nil).Once()

				// Merge call fails
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(555), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, nil, errors.New("merge failed"))
			},
			expectError:   true,
			errorContains: "error merging pull request",
		},
		{
			name:     "nil merge result",
			prNumber: 333,
			setupMock: func(m *mockGithubClient) {
				// Get PR first
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(333)).
					Return(&github.PullRequest{
						Number:    github.Ptr(333),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/333"),
					}, &github.Response{}, nil).Once()

				// Merge returns nil result
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(333), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(nil, &github.Response{}, nil)
			},
			expectError:   true,
			errorContains: "unexpected nil merge result",
		},
		{
			name:     "get PR after merge fails",
			prNumber: 666,
			setupMock: func(m *mockGithubClient) {
				// First Get PR returns mergeable
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(666)).
					Return(&github.PullRequest{
						Number:    github.Ptr(666),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/666"),
					}, &github.Response{}, nil).Once()

				// Merge succeeds
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(666), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("merge_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				// Second Get PR fails
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(666)).
					Return(nil, nil, errors.New("get PR failed")).Once()
			},
			expectError:   true,
			errorContains: "error getting pull request 666 after merge",
		},
		{
			name:     "nil PR returned after merge",
			prNumber: 888,
			setupMock: func(m *mockGithubClient) {
				// First Get PR returns mergeable
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(888)).
					Return(&github.PullRequest{
						Number:    github.Ptr(888),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/888"),
					}, &github.Response{}, nil).Once()

				// Merge succeeds
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(888), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("merge_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				// Second Get PR returns nil
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(888)).
					Return(nil, &github.Response{}, nil).Once()
			},
			expectError:   true,
			errorContains: "unexpected nil pull request after merge",
		},
		{
			name:     "successful merge",
			prNumber: 777,
			setupMock: func(m *mockGithubClient) {
				// First Get PR
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(777)).
					Return(&github.PullRequest{
						Number:    github.Ptr(777),
						State:     github.Ptr("open"),
						Merged:    github.Ptr(false),
						Mergeable: github.Ptr(true),
						Head:      &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:   github.Ptr("https://github.com/akuity/kargo/pull/777"),
					}, &github.Response{}, nil).Once()

				// Merge
				m.On("MergePullRequest", mock.Anything, testRepoOwner, testRepoName, int(777), "",
					mock.AnythingOfType("*github.PullRequestOptions")).
					Return(&github.PullRequestMergeResult{
						SHA:     github.Ptr("merge_sha"),
						Merged:  github.Ptr(true),
						Message: github.Ptr("Pull Request successfully merged"),
					}, &github.Response{}, nil)

				// Second Get PR returns merged
				m.On("GetPullRequests", mock.Anything, testRepoOwner, testRepoName, int(777)).
					Return(&github.PullRequest{
						Number:         github.Ptr(777),
						State:          github.Ptr("closed"),
						Merged:         github.Ptr(true),
						MergeCommitSHA: github.Ptr("merge_sha"),
						Head:           &github.PullRequestBranch{SHA: github.Ptr("head_sha")},
						HTMLURL:        github.Ptr("https://github.com/akuity/kargo/pull/777"),
						MergedAt:       &github.Timestamp{Time: time.Now()},
					}, &github.Response{}, nil).Once()
			},
			expectedMerged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGithubClient{}
			p := provider{
				owner:  testRepoOwner,
				repo:   testRepoName,
				client: mockClient,
			}

			tt.setupMock(mockClient)

			pr, merged, err := p.MergePullRequest(context.Background(), tt.prNumber)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				require.False(t, merged)
				require.Nil(t, pr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedMerged, merged)
				if pr != nil {
					require.Equal(t, tt.prNumber, pr.Number)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetCommitURL(t *testing.T) {
	testCases := []struct {
		repoURL           string
		sha               string
		expectedCommitURL string
	}{
		{
			repoURL:           "ssh://git@github.com/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "git@github.com:akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "https://username@github.com/akuity/kargo",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
		{
			repoURL:           "http://github.com/akuity/kargo.git",
			sha:               "sha",
			expectedCommitURL: "https://github.com/akuity/kargo/commit/sha",
		},
	}

	prov := provider{}

	for _, testCase := range testCases {
		t.Run(testCase.repoURL, func(t *testing.T) {
			commitURL, err := prov.GetCommitURL(testCase.repoURL, testCase.sha)
			require.NoError(t, err)
			require.Equal(t, testCase.expectedCommitURL, commitURL)
		})
	}
}
